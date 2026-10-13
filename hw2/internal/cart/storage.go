package cart

import (
	"sort"
	"sync"
)

// Item это позиция в корзине. Имя и цена копируются из product-service в момент добавления.
type Item struct {
	SKU   int64
	Name  string
	Count uint32
	Price uint32
}

// Storage хранит корзины всех пользователей в памяти.
type Storage struct {
	mu    sync.Mutex
	carts map[int64]map[int64]Item // userID -> sku -> позиция
}

func NewStorage() *Storage {
	return &Storage{carts: make(map[int64]map[int64]Item)}
}

// Add добавляет товар в корзину, при повторном добавлении количество складывается.
func (s *Storage) Add(userID int64, item Item) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cart, ok := s.carts[userID]
	if !ok {
		cart = make(map[int64]Item)
		s.carts[userID] = cart
	}

	if existing, ok := cart[item.SKU]; ok {
		item.Count += existing.Count
	}
	cart[item.SKU] = item
}

// Get возвращает позиции корзины по возрастанию sku и суммарную стоимость.
// Пустой срез означает, что корзины нет.
func (s *Storage) Get(userID int64) ([]Item, uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]Item, 0, len(s.carts[userID]))
	var total uint32
	for _, item := range s.carts[userID] {
		items = append(items, item)
		total += item.Count * item.Price
	}

	// Порядок обхода map случайный, поэтому сортируем явно.
	sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
	return items, total
}
