package cart

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Проверяем контракт GET /user/<user_id>/cart как его видит клиент: порядок items и типы полей в JSON.
func TestGetCartJSONContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     []call
		wantSKUs  []float64
		wantTotal float64
	}{
		{
			name:      "one item",
			setup:     []call{post("/user/1/cart/10", `{"count": 2}`)},
			wantSKUs:  []float64{10},
			wantTotal: 200,
		},
		{
			name: "items come in sku order",
			setup: []call{
				post("/user/1/cart/30", `{"count": 1}`),
				post("/user/1/cart/10", `{"count": 1}`),
				post("/user/1/cart/20", `{"count": 2}`),
			},
			wantSKUs:  []float64{10, 20, 30},
			wantTotal: 600,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakeProducts(copyCatalog(), nil, 0)
			routes := NewHandler(NewStorage(), fake).Routes()
			for _, c := range tc.setup {
				do(routes, c)
			}

			rec := do(routes, get("/user/1/cart"))
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			// Разбираем в map[string]any: числа в JSON становятся float64, строки остаются строками.
			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

			require.IsType(t, float64(0), body["total_price"])
			require.Equal(t, tc.wantTotal, body["total_price"])

			items, ok := body["items"].([]any)
			require.True(t, ok, "items must be an array")
			require.Len(t, items, len(tc.wantSKUs))

			var gotSKUs []float64
			for _, raw := range items {
				item, ok := raw.(map[string]any)
				require.True(t, ok, "item must be an object")
				require.IsType(t, float64(0), item["sku"])
				require.IsType(t, "", item["name"])
				require.IsType(t, float64(0), item["count"])
				require.IsType(t, float64(0), item["price"])
				gotSKUs = append(gotSKUs, item["sku"].(float64))
			}
			require.Equal(t, tc.wantSKUs, gotSKUs)
		})
	}
}
