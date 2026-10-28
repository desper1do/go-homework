//go:build api

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "hw2/api/product"
)

type cartItem struct {
	SKU   int64  `json:"sku"`
	Name  string `json:"name"`
	Count uint32 `json:"count"`
	Price uint32 `json:"price"`
}

type cartBody struct {
	Items      []cartItem `json:"items"`
	TotalPrice uint32     `json:"total_price"`
}

// sendCart выполняет HTTP запрос к cart и возвращает код и тело ответа.
func sendCart(t *testing.T, method, path, body string) (int, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(newContext(t), method, cartURL+path, strings.NewReader(body))
	require.NoError(t, err)

	resp, err := newHTTPClient().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, data
}

func addPath(userID, sku int64) string {
	return fmt.Sprintf("/user/%d/cart/%d", userID, sku)
}

func cartPath(userID int64) string {
	return fmt.Sprintf("/user/%d/cart", userID)
}

// firstProducts берет два первых товара каталога, чтобы не зависеть от конкретных sku.
func firstProducts(t *testing.T, client pb.ProductServiceClient) (*pb.GetProductResponse, *pb.GetProductResponse) {
	t.Helper()

	resp, err := client.ListProducts(newContext(t), &pb.ListProductsRequest{Page: 1, Limit: 2})
	require.NoError(t, err)
	require.Len(t, resp.GetProducts(), 2, "в каталоге должно быть минимум два товара")
	return resp.GetProducts()[0], resp.GetProducts()[1]
}

func TestProductGRPCErrors(t *testing.T) {
	client := dialProduct(t)

	tests := []struct {
		name     string
		sku      int64
		wantCode codes.Code
	}{
		{name: "sku zero", sku: 0, wantCode: codes.InvalidArgument},
		{name: "sku negative", sku: -1, wantCode: codes.InvalidArgument},
		{name: "sku does not exist", sku: math.MaxInt64, wantCode: codes.NotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.GetProduct(newContext(t), &pb.GetProductRequest{Sku: tc.sku})

			require.Equal(t, tc.wantCode, status.Code(err))
		})
	}
}

func TestCartHappyPath(t *testing.T) {
	client := dialProduct(t)
	first, second := firstProducts(t, client)

	got, err := client.GetProduct(newContext(t), &pb.GetProductRequest{Sku: first.GetSku()})
	require.NoError(t, err)
	require.Equal(t, first.GetName(), got.GetName())
	require.Equal(t, first.GetPrice(), got.GetPrice())

	userID := newUserID()

	code, _ := sendCart(t, http.MethodGet, cartPath(userID), "")
	require.Equal(t, http.StatusNotFound, code, "у нового пользователя корзины нет")

	// Порядок добавления обратный порядку sku, повторное добавление складывает количество.
	for _, step := range []struct {
		sku   int64
		count int
	}{{second.GetSku(), 1}, {first.GetSku(), 2}, {first.GetSku(), 3}} {
		code, _ := sendCart(t, http.MethodPost, addPath(userID, step.sku), fmt.Sprintf(`{"count": %d}`, step.count))
		require.Equal(t, http.StatusOK, code)
	}

	code, data := sendCart(t, http.MethodGet, cartPath(userID), "")
	require.Equal(t, http.StatusOK, code)

	var body cartBody
	require.NoError(t, json.Unmarshal(data, &body))

	wantItems := []cartItem{
		{SKU: first.GetSku(), Name: first.GetName(), Count: 5, Price: uint32(first.GetPrice())},
		{SKU: second.GetSku(), Name: second.GetName(), Count: 1, Price: uint32(second.GetPrice())},
	}
	require.Equal(t, wantItems, body.Items)
	require.Equal(t, 5*uint32(first.GetPrice())+uint32(second.GetPrice()), body.TotalPrice)
}

func TestCartHTTPErrors(t *testing.T) {
	client := dialProduct(t)
	first, _ := firstProducts(t, client)

	userID := newUserID()
	unknownSKU := int64(math.MaxInt64)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "add count zero", method: http.MethodPost, path: addPath(userID, first.GetSku()), body: `{"count": 0}`, wantStatus: 400},
		{name: "add count negative", method: http.MethodPost, path: addPath(userID, first.GetSku()), body: `{"count": -1}`, wantStatus: 400},
		{name: "add broken body", method: http.MethodPost, path: addPath(userID, first.GetSku()), body: `{`, wantStatus: 400},
		{name: "add user zero", method: http.MethodPost, path: addPath(0, first.GetSku()), body: `{"count": 1}`, wantStatus: 400},
		{name: "add sku zero", method: http.MethodPost, path: addPath(userID, 0), body: `{"count": 1}`, wantStatus: 400},
		{name: "add unknown product", method: http.MethodPost, path: addPath(userID, unknownSKU), body: `{"count": 1}`, wantStatus: 412},
		{name: "get user zero", method: http.MethodGet, path: cartPath(0), wantStatus: 400},
		{name: "get user without cart", method: http.MethodGet, path: cartPath(userID), wantStatus: 404},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, _ := sendCart(t, tc.method, tc.path, tc.body)

			require.Equal(t, tc.wantStatus, code)
		})
	}
}
