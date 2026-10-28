package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "hw2/api/product"
	"hw2/internal/product"
)

const testCatalog = `{"10": {"name": "ten", "price": 100}}`

// startServer поднимает сервер на свободном порту и возвращает адрес и канал с результатом serve.
// Остановить сервер можно закрытием listener.
func startServer(t *testing.T) (string, net.Listener, chan error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "products.json")
	require.NoError(t, os.WriteFile(path, []byte(testCatalog), 0o600))
	storage, err := product.Load(path)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- serve(lis, storage) }()
	t.Cleanup(func() { _ = lis.Close() })

	return lis.Addr().String(), lis, done
}

func TestServe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sku      int64
		wantCode codes.Code
		wantName string
	}{
		{name: "existing product", sku: 10, wantCode: codes.OK, wantName: "ten"},
		{name: "unknown product", sku: 11, wantCode: codes.NotFound},
		{name: "bad sku", sku: 0, wantCode: codes.InvalidArgument},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			addr, _, _ := startServer(t)
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			resp, err := pb.NewProductServiceClient(conn).GetProduct(ctx, &pb.GetProductRequest{Sku: tc.sku})

			require.Equal(t, tc.wantCode, status.Code(err))
			require.Equal(t, tc.wantName, resp.GetName())
		})
	}
}

func TestServeStopsWhenListenerClosed(t *testing.T) {
	t.Parallel()

	_, lis, done := startServer(t)
	require.NoError(t, lis.Close())

	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		require.Fail(t, "serve did not return after listener was closed")
	}
}
