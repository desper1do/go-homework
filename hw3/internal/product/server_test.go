package product

import (
	"context"
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "hw2/api/product"
)

// newTestServer собирает сервис поверх каталога из переданной map, минуя чтение файла.
func newTestServer(products map[int64]Product) *Server {
	skus := make([]int64, 0, len(products))
	for sku := range products {
		skus = append(skus, sku)
	}
	slices.Sort(skus)
	return NewServer(&Storage{products: products, skus: skus})
}

// catalog создает n товаров с sku 10, 20, ..., 10*n.
func catalog(n int) map[int64]Product {
	products := make(map[int64]Product, n)
	for i := 1; i <= n; i++ {
		products[int64(i*10)] = Product{Name: "item", Price: uint32(i)}
	}
	return products
}

// skuRange возвращает ожидаемые sku каталога из catalog для номеров from..to включительно.
func skuRange(from, to int) []int64 {
	var skus []int64
	for i := from; i <= to; i++ {
		skus = append(skus, int64(i*10))
	}
	return skus
}

func listedSKUs(resp *pb.ListProductsResponse) []int64 {
	var skus []int64
	for _, p := range resp.GetProducts() {
		skus = append(skus, p.GetSku())
	}
	return skus
}

func TestGetProduct(t *testing.T) {
	t.Parallel()

	products := map[int64]Product{
		7:  {Name: "book", Price: 100},
		8:  {Name: "free", Price: 0},
		9:  {Name: "max", Price: math.MaxInt32},
		10: {Name: "too expensive", Price: math.MaxInt32 + 1},
		11: {Name: "huge", Price: math.MaxUint32},
	}

	tests := []struct {
		name     string
		sku      int64
		wantCode codes.Code
		want     *pb.GetProductResponse
	}{
		{name: "found", sku: 7, wantCode: codes.OK, want: &pb.GetProductResponse{Sku: 7, Name: "book", Price: 100}},
		{name: "zero price", sku: 8, wantCode: codes.OK, want: &pb.GetProductResponse{Sku: 8, Name: "free", Price: 0}},
		{name: "max int32 price", sku: 9, wantCode: codes.OK, want: &pb.GetProductResponse{Sku: 9, Name: "max", Price: math.MaxInt32}},
		{name: "price overflows int32", sku: 10, wantCode: codes.Internal},
		{name: "price is max uint32", sku: 11, wantCode: codes.Internal},
		{name: "not found", sku: 12, wantCode: codes.NotFound},
		{name: "sku zero", sku: 0, wantCode: codes.InvalidArgument},
		{name: "sku negative", sku: -7, wantCode: codes.InvalidArgument},
		{name: "sku min int64", sku: math.MinInt64, wantCode: codes.InvalidArgument},
		{name: "sku max int64 not found", sku: math.MaxInt64, wantCode: codes.NotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(maps(products))
			resp, err := srv.GetProduct(context.Background(), &pb.GetProductRequest{Sku: tc.sku})

			require.Equal(t, tc.wantCode, status.Code(err))
			require.True(t, proto.Equal(tc.want, resp), "got %v, want %v", resp, tc.want)
		})
	}
}

func TestGetProductEmptyCatalog(t *testing.T) {
	t.Parallel()

	srv := newTestServer(map[int64]Product{})
	_, err := srv.GetProduct(context.Background(), &pb.GetProductRequest{Sku: 1})

	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestListProducts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		products  map[int64]Product
		page      int32
		limit     int32
		wantCode  codes.Code
		wantSKUs  []int64
		wantTotal int32
	}{
		{name: "defaults give first ten", products: catalog(12), wantCode: codes.OK, wantSKUs: skuRange(1, 10), wantTotal: 12},
		{name: "page zero is first page", products: catalog(5), page: 0, limit: 2, wantCode: codes.OK, wantSKUs: skuRange(1, 2), wantTotal: 5},
		{name: "second page", products: catalog(5), page: 2, limit: 2, wantCode: codes.OK, wantSKUs: skuRange(3, 4), wantTotal: 5},
		{name: "last page is partial", products: catalog(5), page: 3, limit: 2, wantCode: codes.OK, wantSKUs: skuRange(5, 5), wantTotal: 5},
		{name: "page past the end", products: catalog(5), page: 4, limit: 2, wantCode: codes.OK, wantSKUs: nil, wantTotal: 5},
		{name: "limit larger than catalog", products: catalog(3), limit: 100, wantCode: codes.OK, wantSKUs: skuRange(1, 3), wantTotal: 3},
		{name: "empty catalog", products: catalog(0), wantCode: codes.OK, wantSKUs: nil, wantTotal: 0},
		{name: "huge page does not overflow", products: catalog(3), page: math.MaxInt32, limit: 100, wantCode: codes.OK, wantSKUs: nil, wantTotal: 3},
		{name: "negative page", products: catalog(3), page: -1, wantCode: codes.InvalidArgument},
		{name: "negative limit", products: catalog(3), limit: -1, wantCode: codes.InvalidArgument},
		{name: "limit over max", products: catalog(3), limit: 101, wantCode: codes.InvalidArgument},
		{
			name:      "price overflow fails the page",
			products:  map[int64]Product{1: {Name: "big", Price: math.MaxUint32}},
			wantCode:  codes.Internal,
			wantTotal: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := newTestServer(maps(tc.products))
			resp, err := srv.ListProducts(context.Background(), &pb.ListProductsRequest{Page: tc.page, Limit: tc.limit})

			require.Equal(t, tc.wantCode, status.Code(err))
			require.Equal(t, tc.wantSKUs, listedSKUs(resp))
			require.Equal(t, tc.wantTotal, resp.GetTotal())
		})
	}
}

// maps копирует каталог, чтобы у каждого кейса была своя map.
func maps(src map[int64]Product) map[int64]Product {
	dst := make(map[int64]Product, len(src))
	for sku, p := range src {
		dst[sku] = p
	}
	return dst
}
