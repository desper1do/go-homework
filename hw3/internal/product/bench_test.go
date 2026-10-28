package product

import (
	"context"
	"testing"

	pb "hw2/api/product"
)

func BenchmarkGetProduct(b *testing.B) {
	// Сервис и запрос создаем до таймера, измеряем только сам вызов.
	srv := newTestServer(catalog(1000))
	req := &pb.GetProductRequest{Sku: 500 * 10}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := srv.GetProduct(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}
