package cart

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// quietLogs выключает лог middleware, иначе вывод бенчмарка тонет в строках запросов.
func quietLogs(b *testing.B) {
	b.Helper()
	old := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(old) })
}

func BenchmarkAddToCart(b *testing.B) {
	quietLogs(b)
	routes := NewHandler(NewStorage(), newFakeProducts(copyCatalog(), nil, 0)).Routes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/user/1/cart/10", strings.NewReader(`{"count": 1}`))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status %d", rec.Code)
		}
	}
}

func BenchmarkGetCart(b *testing.B) {
	quietLogs(b)
	routes := NewHandler(NewStorage(), newFakeProducts(copyCatalog(), nil, 0)).Routes()

	// Заранее кладем в корзину три товара.
	for _, sku := range []string{"10", "20", "30"} {
		req := httptest.NewRequest(http.MethodPost, "/user/1/cart/"+sku, strings.NewReader(`{"count": 1}`))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("setup failed with status %d", rec.Code)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/user/1/cart", nil)
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status %d", rec.Code)
		}
	}
}

// Дальше бенчмарки самого хранилища, чтобы сравнивать цену блокировок.
// RunParallel запускает тело в нескольких горутинах сразу, как параллельные HTTP запросы.

func BenchmarkStorageGetParallel(b *testing.B) {
	s := NewStorage()
	for sku := int64(1); sku <= 10; sku++ {
		s.Add(1, sku, 1)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Get(1)
		}
	})
}

func BenchmarkStorageAddParallel(b *testing.B) {
	s := NewStorage()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var sku int64
		for pb.Next() {
			sku = sku%10 + 1
			s.Add(1, sku, 1)
		}
	})
}
