package http

import (
	"encoding/json"
	"net/http"

	"gigpurse/internal/domain"
)

type WalletHandler struct {
	walletUsecase domain.WalletUsecase
}

func NewWalletHandler(wu domain.WalletUsecase) *WalletHandler {
	return &WalletHandler{
		walletUsecase: wu,
	}
}

func (h *WalletHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/wallet", h.HandleWallet)
	mux.HandleFunc("/wallet/deposit", h.HandleDeposit)
}

func (h *WalletHandler) HandleWallet(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetBalance(w, r)
	case http.MethodPost:
		h.CreateWallet(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	balance, err := h.walletUsecase.GetBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"balance": balance,
	})
}

func (h *WalletHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.walletUsecase.CreateWallet(r.Context(), req.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "wallet created successfully",
	})
}

func (h *WalletHandler) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string  `json:"user_id"`
		Amount float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.walletUsecase.Deposit(r.Context(), req.UserID, req.Amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "deposit successful",
	})
}
