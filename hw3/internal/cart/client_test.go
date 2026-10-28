package cart

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "hw2/api/product"
)

var errBoom = errors.New("boom")

// stubGRPC подменяет gRPC клиент product-service, нужный метод встроенный интерфейс не реализует.
type stubGRPC struct {
	pb.ProductServiceClient
	resp *pb.GetProductResponse
	err  error
}

func (s stubGRPC) GetProduct(context.Context, *pb.GetProductRequest, ...grpc.CallOption) (*pb.GetProductResponse, error) {
	return s.resp, s.err
}

func TestProductClientGetProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resp       *pb.GetProductResponse
		err        error
		wantName   string
		wantPrice  uint32
		wantFailed bool
		wantIs     error
	}{
		{name: "found", resp: &pb.GetProductResponse{Name: "book", Price: 100}, wantName: "book", wantPrice: 100},
		{name: "zero price", resp: &pb.GetProductResponse{Name: "free"}, wantName: "free"},
		{name: "not found maps to sentinel", err: status.Error(codes.NotFound, "x"), wantFailed: true, wantIs: ErrProductNotFound},
		{name: "other error is wrapped", err: errBoom, wantFailed: true, wantIs: errBoom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := NewProductClient(stubGRPC{resp: tc.resp, err: tc.err})
			name, price, err := client.GetProduct(context.Background(), 1)

			require.Equal(t, tc.wantFailed, err != nil)
			require.ErrorIs(t, err, tc.wantIs)
			require.Equal(t, tc.wantName, name)
			require.Equal(t, tc.wantPrice, price)
		})
	}
}

func TestProductClientNegativePrice(t *testing.T) {
	t.Parallel()

	client := NewProductClient(stubGRPC{resp: &pb.GetProductResponse{Name: "bad", Price: -1}})
	_, _, err := client.GetProduct(context.Background(), 1)

	require.Error(t, err)
}
