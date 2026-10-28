package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "hw2/api/product"
	"hw2/internal/product"
)

const testCatalog = `{"10": {"name": "ten", "price": 100}}`

// startProduct поднимает маленький настоящий product-service на свободном порту.
func startProduct(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "products.json")
	require.NoError(t, os.WriteFile(path, []byte(testCatalog), 0o600))
	storage, err := product.Load(path)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	pb.RegisterProductServiceServer(srv, product.NewServer(storage))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return lis.Addr().String()
}

// startCart поднимает cart на свободном порту, он ходит в product-service по productAddr.
func startCart(t *testing.T, productAddr string) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() { _ = serve(lis, productAddr) }()
	t.Cleanup(func() { _ = lis.Close() })

	return "http://" + lis.Addr().String()
}

func request(t *testing.T, method, url, body string) (int, []byte) {
	t.Helper()

	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, data
}

// Мини сквозной тест: HTTP в cart, потом настоящий gRPC в product-service.
func TestServeEndToEnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		addPath       string
		wantAddStatus int
		wantGetStatus int
		wantTotal     uint32
	}{
		{name: "add existing product", addPath: "/user/1/cart/10", wantAddStatus: 200, wantGetStatus: 200, wantTotal: 200},
		{name: "add unknown product", addPath: "/user/1/cart/11", wantAddStatus: 412, wantGetStatus: 404, wantTotal: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := startCart(t, startProduct(t))

			code, _ := request(t, http.MethodPost, base+tc.addPath, `{"count": 2}`)
			require.Equal(t, tc.wantAddStatus, code)

			code, data := request(t, http.MethodGet, base+"/user/1/cart", "")
			require.Equal(t, tc.wantGetStatus, code)

			// Для 404 тело это текст, разбор не получится и total останется нулем.
			var body struct {
				TotalPrice uint32 `json:"total_price"`
			}
			_ = json.Unmarshal(data, &body)
			require.Equal(t, tc.wantTotal, body.TotalPrice)
		})
	}
}
