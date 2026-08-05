package http

import (
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
