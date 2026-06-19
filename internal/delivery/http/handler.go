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
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "missing_user_id", "user_id query parameter is required")
		return
	}

	balance, err := h.walletUsecase.GetBalance(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "wallet_not_found", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "wallet balance retrieved successfully", map[string]interface{}{
		"user_id": userID,
		"balance": balance,
	})
}

func (h *WalletHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	if err := h.walletUsecase.CreateWallet(r.Context(), req.UserID); err != nil {
		respondError(w, http.StatusInternalServerError, "wallet_create_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusCreated, "wallet created successfully", map[string]string{
		"message": "wallet created successfully",
	})
}

func (h *WalletHandler) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req struct {
		UserID string  `json:"user_id"`
		Amount float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	err := h.walletUsecase.Deposit(r.Context(), req.UserID, req.Amount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "wallet_deposit_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "deposit successful", map[string]string{
		"message": "deposit successful",
	})
}
