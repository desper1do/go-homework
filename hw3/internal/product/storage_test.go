package product

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantFailed bool
		wantSKUs   []int64
	}{
		{name: "sorted by sku", content: `{"30": {"name": "c", "price": 3}, "5": {"name": "a", "price": 1}, "100": {"name": "b", "price": 2}}`, wantSKUs: []int64{5, 30, 100}},
		{name: "empty object", content: `{}`, wantSKUs: []int64{}},
		{name: "broken json", content: `{`, wantFailed: true},
		{name: "sku is not a number", content: `{"abc": {"name": "a", "price": 1}}`, wantFailed: true},
		{name: "negative price", content: `{"1": {"name": "a", "price": -1}}`, wantFailed: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "products.json")
			require.NoError(t, os.WriteFile(path, []byte(tc.content), 0o600))

			storage, err := Load(path)

			require.Equal(t, tc.wantFailed, err != nil)
			require.Equal(t, tc.wantSKUs, storage.skusOrNil())
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	_, err := Load(filepath.Join(t.TempDir(), "absent.json"))

	require.ErrorIs(t, err, os.ErrNotExist)
}

// skusOrNil нужен только тестам: для nil хранилища возвращает nil, чтобы проверка не ветвилась.
func (s *Storage) skusOrNil() []int64 {
	if s == nil {
		return nil
	}
	return s.skus
}
