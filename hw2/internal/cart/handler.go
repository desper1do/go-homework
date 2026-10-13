package cart

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
)

// Handler обслуживает HTTP API корзины.
type Handler struct {
	storage  *Storage
	products *ProductClient
}

func NewHandler(storage *Storage, products *ProductClient) *Handler {
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

	name, price, err := h.products.GetProduct(r.Context(), sku)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			http.Error(w, "product does not exist", http.StatusPreconditionFailed)
			return
		}
		log.Printf("add item: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.storage.Add(userID, Item{SKU: sku, Name: name, Count: uint32(req.Count), Price: price})
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getCart(w http.ResponseWriter, r *http.Request) {
	userID, err := positiveID(r.PathValue("user_id"))
	if err != nil {
		http.Error(w, "user_id must be a positive integer", http.StatusBadRequest)
		return
	}

	items, total := h.storage.Get(userID)
	if len(items) == 0 {
		http.Error(w, "cart not found", http.StatusNotFound)
		return
	}

	resp := cartResponse{Items: make([]itemResponse, 0, len(items)), TotalPrice: total}
	for _, item := range items {
		resp.Items = append(resp.Items, itemResponse(item))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("get cart: write response: %v", err)
	}
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
