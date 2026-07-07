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
	mux.HandleFunc("/wallet", JWTMiddleware(h.GetWallet))
	mux.HandleFunc("/wallet/deposit", JWTMiddleware(h.Deposit))
	mux.HandleFunc("/wallet/withdraw", JWTMiddleware(h.Withdraw))
	mux.HandleFunc("/wallet/transactions", JWTMiddleware(h.ListTransactions))
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	wallet, err := h.walletUsecase.GetWallet(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "wallet_fetch_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "wallet retrieved successfully", wallet)
}

func (h *WalletHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	wallet, err := h.walletUsecase.Deposit(r.Context(), userID, req.Amount)
	if err != nil {
		respondError(w, http.StatusBadRequest, "wallet_deposit_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "deposit successful", wallet)
}

func (h *WalletHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	wallet, err := h.walletUsecase.Withdraw(r.Context(), userID, req.Amount)
	if err != nil {
		respondError(w, http.StatusBadRequest, "wallet_withdraw_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "withdrawal successful", wallet)
}

func (h *WalletHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	txs, err := h.walletUsecase.ListTransactions(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "transactions_fetch_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "transactions retrieved successfully", txs)
}
