package cart

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// productTimeout ограничивает все обращения к product-service в рамках одного HTTP запроса.
const productTimeout = 3 * time.Second

// ProductGetter это то, что cart требует от клиента product-service.
// Интерфейс объявлен здесь, у потребителя, чтобы в тестах подменять клиента своей реализацией.
type ProductGetter interface {
	GetProduct(ctx context.Context, sku int64) (name string, price uint32, err error)
}

// Handler обслуживает HTTP API корзины.
type Handler struct {
	storage  *Storage
	products ProductGetter
}

func NewHandler(storage *Storage, products ProductGetter) *Handler {
	return &Handler{storage: storage, products: products}
}

// Routes регистрирует маршруты, параметры пути берутся через r.PathValue.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /user/{user_id}/cart/{sku_id}", h.addItem)
	mux.HandleFunc("GET /user/{user_id}/cart", h.getCart)
	mux.Handle("GET /swagger/", swaggerHandler())
	return LoggingMiddleware(mux)
}

type addItemRequest struct {
	// int64 вместо uint32, чтобы отрицательное значение дало 400 с понятной причиной,
	// а не ошибку разбора JSON.
	Count int64 `json:"count"`
}

type itemResponse struct {
	SKU   int64  `json:"sku"`
	Name  string `json:"name"`
	Count uint32 `json:"count"`
	Price uint32 `json:"price"`
}

type cartResponse struct {
	Items      []itemResponse `json:"items"`
	TotalPrice uint32         `json:"total_price"`
}

func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) {
	userID, err := positiveID(r.PathValue("user_id"))
	if err != nil {
		http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
		return
	}
	sku, err := positiveID(r.PathValue("sku_id"))
	if err != nil {
		http.Error(w, "sku_id must be a positive integer", http.StatusBadRequest)
		return
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Count <= 0 || req.Count > math.MaxUint32 {
		http.Error(w, "count must be a positive integer", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), productTimeout)
	defer cancel()

	_, _, err = h.products.GetProduct(ctx, sku)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			http.Error(w, "product does not exist", http.StatusPreconditionFailed)
			return
		}
		log.Printf("add item: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.storage.Add(userID, sku, uint32(req.Count))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getCart(w http.ResponseWriter, r *http.Request) {
	userID, err := positiveID(r.PathValue("user_id"))
	if err != nil {
		http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
		return
	}

	entries := h.storage.Get(userID)
	if len(entries) == 0 {
		http.Error(w, "cart not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), productTimeout)
	defer cancel()

	items, err := h.fetchItems(ctx, entries)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			http.Error(w, "product does not exist", http.StatusPreconditionFailed)
			return
		}
		log.Printf("get cart: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := cartResponse{Items: items}
	for _, item := range items {
		resp.TotalPrice += item.Count * item.Price
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("get cart: write response: %v", err)
	}
}

// fetchItems запрашивает описание каждого товара в отдельной горутине.
// Каждая горутина пишет только в свои ячейки срезов по своему индексу, поэтому мьютекс не нужен.
// Если хоть один запрос упал, возвращается ошибка первого по порядку sku, частичный результат не отдается.
func (h *Handler) fetchItems(ctx context.Context, entries []Entry) ([]itemResponse, error) {
	items := make([]itemResponse, len(entries))
	errs := make([]error, len(entries))

	var wg sync.WaitGroup
	for i, entry := range entries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name, price, err := h.products.GetProduct(ctx, entry.SKU)
			items[i] = itemResponse{SKU: entry.SKU, Name: name, Count: entry.Count, Price: price}
			errs[i] = err
		}()
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

// positiveID разбирает строку в натуральное число (больше нуля).
func positiveID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, errors.New("id must be positive")
	}
	return id, nil
}
