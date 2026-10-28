package cart

import (
	"sort"
	"sync"
)

// Entry это позиция в корзине. Имя и цену корзина не хранит,
// они запрашиваются из product-service при чтении.
type Entry struct {
	SKU   int64
	Count uint32
}

// Storage хранит корзины всех пользователей в памяти.
// Чтение идет под RLock, запись под Lock. Счетчика, который менялся бы одной
// операцией, здесь нет, поэтому atomic не используется.
type Storage struct {
	mu    sync.RWMutex
	carts map[int64]map[int64]uint32 // userID -> sku -> количество
}

func NewStorage() *Storage {
	return &Storage{carts: make(map[int64]map[int64]uint32)}
}

// Add добавляет товар в корзину, при повторном добавлении количество складывается.
func (s *Storage) Add(userID, sku int64, count uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cart, ok := s.carts[userID]
	if !ok {
		cart = make(map[int64]uint32)
		s.carts[userID] = cart
	}
	cart[sku] += count
}

// Get возвращает позиции корзины по возрастанию sku.
// Пустой срез означает, что корзины нет.
func (s *Storage) Get(userID int64) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]Entry, 0, len(s.carts[userID]))
	for sku, count := range s.carts[userID] {
		entries = append(entries, Entry{SKU: sku, Count: count})
	}

	// Порядок обхода map случайный, поэтому сортируем явно.
	sort.Slice(entries, func(i, j int) bool { return entries[i].SKU < entries[j].SKU })
	return entries
}
