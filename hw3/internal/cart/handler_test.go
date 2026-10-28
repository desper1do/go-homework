package cart

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

var errNoDeadline = errors.New("outgoing call without deadline")

type fakeProduct struct {
	name  string
	price uint32
}

// fakeProducts это тестовый двойник клиента product-service.
// Он считает вызовы, требует дедлайн у контекста и умеет ждать, пока
// в него одновременно придет barrier вызовов, это доказывает параллельность.
type fakeProducts struct {
	mu       sync.Mutex
	products map[int64]fakeProduct
	errs     map[int64]error
	barrier  int
	arrived  int
	released chan struct{}
	calls    atomic.Int32
}

func newFakeProducts(products map[int64]fakeProduct, errs map[int64]error, barrier int) *fakeProducts {
	return &fakeProducts{products: products, errs: errs, barrier: barrier, released: make(chan struct{})}
}

func (f *fakeProducts) remove(skus ...int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, sku := range skus {
		delete(f.products, sku)
	}
}

func (f *fakeProducts) GetProduct(ctx context.Context, sku int64) (string, uint32, error) {
	f.calls.Add(1)
	if _, ok := ctx.Deadline(); !ok {
		return "", 0, errNoDeadline
	}
	if err := f.waitBarrier(ctx); err != nil {
		return "", 0, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.errs[sku]; err != nil {
		return "", 0, err
	}
	p, ok := f.products[sku]
	if !ok {
		return "", 0, ErrProductNotFound
	}
	return p.name, p.price, nil
}

func (f *fakeProducts) waitBarrier(ctx context.Context) error {
	if f.barrier == 0 {
		return nil
	}
	f.mu.Lock()
	f.arrived++
	if f.arrived == f.barrier {
		close(f.released)
	}
	f.mu.Unlock()

	select {
	case <-f.released:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type call struct {
	method, path, body string
}

func post(path, body string) call { return call{http.MethodPost, path, body} }
func get(path string) call        { return call{http.MethodGet, path, ""} }

func do(h http.Handler, c call) *httptest.ResponseRecorder {
	req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeCart(rec *httptest.ResponseRecorder) cartResponse {
	var resp cartResponse
	if rec.Code == http.StatusOK {
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	}
	return resp
}

var catalogFake = map[int64]fakeProduct{
	10: {name: "ten", price: 100},
	20: {name: "twenty", price: 250},
	30: {name: "thirty", price: 0},
}

func copyCatalog() map[int64]fakeProduct {
	dst := make(map[int64]fakeProduct, len(catalogFake))
	for sku, p := range catalogFake {
		dst[sku] = p
	}
	return dst
}

func TestAddItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		errs        map[int64]error
		setup       []call
		request     call
		wantStatus  int
		wantEntries []Entry
		wantCalls   int32
	}{
		{name: "ok", request: post("/user/1/cart/10", `{"count": 2}`), wantStatus: 200, wantEntries: []Entry{{10, 2}}, wantCalls: 1},
		{name: "max count", request: post("/user/1/cart/10", `{"count": 4294967295}`), wantStatus: 200, wantEntries: []Entry{{10, 4294967295}}, wantCalls: 1},
		{
			name:        "repeated add sums count",
			setup:       []call{post("/user/1/cart/10", `{"count": 2}`)},
			request:     post("/user/1/cart/10", `{"count": 3}`),
			wantStatus:  200,
			wantEntries: []Entry{{10, 5}},
			wantCalls:   1,
		},
		{
			name:        "different sku",
			setup:       []call{post("/user/1/cart/20", `{"count": 1}`)},
			request:     post("/user/1/cart/10", `{"count": 1}`),
			wantStatus:  200,
			wantEntries: []Entry{{10, 1}, {20, 1}},
			wantCalls:   1,
		},
		{name: "count over uint32", request: post("/user/1/cart/10", `{"count": 4294967296}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "count zero", request: post("/user/1/cart/10", `{"count": 0}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "count negative", request: post("/user/1/cart/10", `{"count": -1}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "count missing", request: post("/user/1/cart/10", `{}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "count is a string", request: post("/user/1/cart/10", `{"count": "2"}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "count is fractional", request: post("/user/1/cart/10", `{"count": 1.5}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "broken json", request: post("/user/1/cart/10", `{`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "empty body", request: post("/user/1/cart/10", ``), wantStatus: 400, wantEntries: []Entry{}},
		{name: "user zero", request: post("/user/0/cart/10", `{"count": 1}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "user negative", request: post("/user/-1/cart/10", `{"count": 1}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "user not a number", request: post("/user/abc/cart/10", `{"count": 1}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "sku zero", request: post("/user/1/cart/0", `{"count": 1}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "sku negative", request: post("/user/1/cart/-5", `{"count": 1}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "sku overflows int64", request: post("/user/1/cart/99999999999999999999", `{"count": 1}`), wantStatus: 400, wantEntries: []Entry{}},
		{name: "product does not exist", request: post("/user/1/cart/999", `{"count": 1}`), wantStatus: 412, wantEntries: []Entry{}, wantCalls: 1},
		{
			name:        "product service failure",
			errs:        map[int64]error{10: errBoom},
			request:     post("/user/1/cart/10", `{"count": 1}`),
			wantStatus:  500,
			wantEntries: []Entry{},
			wantCalls:   1,
		},
		{
			name:        "failed add keeps earlier items",
			setup:       []call{post("/user/1/cart/10", `{"count": 1}`)},
			request:     post("/user/1/cart/999", `{"count": 1}`),
			wantStatus:  412,
			wantEntries: []Entry{{10, 1}},
			wantCalls:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakeProducts(copyCatalog(), tc.errs, 0)
			h := NewHandler(NewStorage(), fake)
			routes := h.Routes()
			for _, c := range tc.setup {
				do(routes, c)
			}
			fake.calls.Store(0)

			rec := do(routes, tc.request)

			require.Equal(t, tc.wantStatus, rec.Code)
			require.Equal(t, tc.wantEntries, h.storage.Get(1))
			require.Equal(t, tc.wantCalls, fake.calls.Load())
		})
	}
}

func TestGetCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		errs       map[int64]error
		barrier    int
		setup      []call
		vanish     []int64
		request    call
		wantStatus int
		want       cartResponse
		wantCalls  int32
	}{
		{name: "empty cart is 404 without product calls", request: get("/user/1/cart"), wantStatus: 404},
		{
			name:       "single item",
			setup:      []call{post("/user/1/cart/10", `{"count": 2}`)},
			request:    get("/user/1/cart"),
			wantStatus: 200,
			want:       cartResponse{Items: []itemResponse{{SKU: 10, Name: "ten", Count: 2, Price: 100}}, TotalPrice: 200},
			wantCalls:  1,
		},
		{
			name: "items sorted by sku with total",
			setup: []call{
				post("/user/1/cart/30", `{"count": 1}`),
				post("/user/1/cart/10", `{"count": 1}`),
				post("/user/1/cart/20", `{"count": 2}`),
			},
			request:    get("/user/1/cart"),
			wantStatus: 200,
			want: cartResponse{
				Items: []itemResponse{
					{SKU: 10, Name: "ten", Count: 1, Price: 100},
					{SKU: 20, Name: "twenty", Count: 2, Price: 250},
					{SKU: 30, Name: "thirty", Count: 1, Price: 0},
				},
				TotalPrice: 600,
			},
			wantCalls: 3,
		},
		{
			name:       "repeated add sums count in response",
			setup:      []call{post("/user/1/cart/10", `{"count": 2}`), post("/user/1/cart/10", `{"count": 3}`)},
			request:    get("/user/1/cart"),
			wantStatus: 200,
			want:       cartResponse{Items: []itemResponse{{SKU: 10, Name: "ten", Count: 5, Price: 100}}, TotalPrice: 500},
			wantCalls:  1,
		},
		{
			name:       "only free items give zero total",
			setup:      []call{post("/user/1/cart/30", `{"count": 4}`)},
			request:    get("/user/1/cart"),
			wantStatus: 200,
			want:       cartResponse{Items: []itemResponse{{SKU: 30, Name: "thirty", Count: 4, Price: 0}}, TotalPrice: 0},
			wantCalls:  1,
		},
		{
			name:       "other user cart is not visible",
			setup:      []call{post("/user/2/cart/10", `{"count": 1}`)},
			request:    get("/user/1/cart"),
			wantStatus: 404,
			wantCalls:  0,
		},
		{
			name: "product requests run in parallel",
			setup: []call{
				post("/user/1/cart/10", `{"count": 1}`),
				post("/user/1/cart/20", `{"count": 1}`),
				post("/user/1/cart/30", `{"count": 1}`),
			},
			// barrier срабатывает только когда три GetProduct ждут одновременно,
			// при последовательных вызовах запрос упал бы по таймауту.
			barrier:    3,
			request:    get("/user/1/cart"),
			wantStatus: 200,
			want: cartResponse{
				Items: []itemResponse{
					{SKU: 10, Name: "ten", Count: 1, Price: 100},
					{SKU: 20, Name: "twenty", Count: 1, Price: 250},
					{SKU: 30, Name: "thirty", Count: 1, Price: 0},
				},
				TotalPrice: 350,
			},
			wantCalls: 3,
		},
		{
			name:       "product vanished gives 412 without partial answer",
			setup:      []call{post("/user/1/cart/10", `{"count": 1}`), post("/user/1/cart/20", `{"count": 1}`)},
			vanish:     []int64{20},
			request:    get("/user/1/cart"),
			wantStatus: 412,
			wantCalls:  2,
		},
		{name: "user zero", request: get("/user/0/cart"), wantStatus: 400},
		{name: "user negative", request: get("/user/-3/cart"), wantStatus: 400},
		{name: "user not a number", request: get("/user/abc/cart"), wantStatus: 400},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakeProducts(copyCatalog(), tc.errs, 0)
			routes := NewHandler(NewStorage(), fake).Routes()
			for _, c := range tc.setup {
				do(routes, c)
			}
			fake.calls.Store(0)
			fake.remove(tc.vanish...)
			fake.setBarrier(tc.barrier)

			rec := do(routes, tc.request)

			require.Equal(t, tc.wantStatus, rec.Code)
			require.Equal(t, tc.want, decodeCart(rec))
			require.Equal(t, tc.wantCalls, fake.calls.Load())
		})
	}
}

func (f *fakeProducts) setBarrier(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.barrier = n
}

// TestGetCartProductFailure отдельно проверяет 500: ошибка не NotFound не должна превращаться в 412.
func TestGetCartProductFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		failErr    error
		wantStatus int
	}{
		{name: "generic error is 500", failErr: errBoom, wantStatus: 500},
		{name: "context error is 500", failErr: context.DeadlineExceeded, wantStatus: 500},
		{name: "not found is 412", failErr: ErrProductNotFound, wantStatus: 412},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakeProducts(copyCatalog(), nil, 0)
			routes := NewHandler(NewStorage(), fake).Routes()
			do(routes, post("/user/1/cart/10", `{"count": 1}`))
			do(routes, post("/user/1/cart/20", `{"count": 1}`))
			fake.setErr(20, tc.failErr)

			rec := do(routes, get("/user/1/cart"))

			require.Equal(t, tc.wantStatus, rec.Code)
			require.Equal(t, cartResponse{}, decodeCart(rec))
		})
	}
}

func (f *fakeProducts) setErr(sku int64, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs = map[int64]error{sku: err}
}
