package product

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

// Product это товар из каталога. Цена внутри сервисов хранится как uint32.
type Product struct {
	Name  string `json:"name"`
	Price uint32 `json:"price"`
}

// Item это товар вместе со своим sku.
type Item struct {
	SKU int64
	Product
}

// Storage хранит каталог товаров в памяти. После загрузки он только читается,
// поэтому мьютекс не нужен.
type Storage struct {
	products map[int64]Product
	// skus отсортирован по возрастанию и строится один раз при загрузке,
	// чтобы порядок в постраничном выводе не зависел от порядка обхода map.
	skus []int64
}

// Load читает каталог из JSON файла вида {"<sku>": {"name": ..., "price": ...}}.
func Load(path string) (*Storage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read products file: %w", err)
	}

	// Ключи JSON всегда строки, encoding/json сам разбирает их в int64.
	var products map[int64]Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, fmt.Errorf("parse products file: %w", err)
	}

	skus := make([]int64, 0, len(products))
	for sku := range products {
		skus = append(skus, sku)
	}
	slices.Sort(skus)

	return &Storage{products: products, skus: skus}, nil
}

// Get возвращает товар по sku, второе значение false если товара нет.
func (s *Storage) Get(sku int64) (Product, bool) {
	p, ok := s.products[sku]
	return p, ok
}

// Total возвращает общее число товаров в каталоге.
func (s *Storage) Total() int {
	return len(s.skus)
}

// List возвращает не больше limit товаров начиная с позиции offset в порядке возрастания sku.
// Если offset за концом каталога, результат пустой.
func (s *Storage) List(offset, limit int) []Item {
	if offset >= len(s.skus) {
		return nil
	}
	end := min(offset+limit, len(s.skus))

	items := make([]Item, 0, end-offset)
	for _, sku := range s.skus[offset:end] {
		items = append(items, Item{SKU: sku, Product: s.products[sku]})
	}
	return items
}
