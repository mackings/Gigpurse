package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"gigpurse/internal/usecase"
)

type PayoutAccountHandler struct {
	payoutUsecase usecase.PayoutAccountUsecase
}

func NewPayoutAccountHandler(pu usecase.PayoutAccountUsecase) *PayoutAccountHandler {
	return &PayoutAccountHandler{payoutUsecase: pu}
}

func (h *PayoutAccountHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/payout-account/banks", JWTMiddleware(h.ListBanks))
	mux.HandleFunc("/payout-account/validate", JWTMiddleware(h.Validate))
	mux.HandleFunc("/payout-account", JWTMiddleware(h.Link))
}

func (h *PayoutAccountHandler) ListBanks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if _, _, ok := GetUserFromContext(r.Context()); !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	banks, err := h.payoutUsecase.ListBanks(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, "banks_fetch_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "banks retrieved successfully", banks)
}

func (h *PayoutAccountHandler) Validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if _, _, ok := GetUserFromContext(r.Context()); !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req struct {
		BankCode      string `json:"bank_code"`
		AccountNumber string `json:"account_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	accountName, err := h.payoutUsecase.Validate(r.Context(), req.BankCode, req.AccountNumber)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bank_validation_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "account validated successfully", map[string]string{"account_name": accountName})
}

func (h *PayoutAccountHandler) Link(w http.ResponseWriter, r *http.Request) {
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
		BankCode      string `json:"bank_code"`
		BankName      string `json:"bank_name"`
		AccountNumber string `json:"account_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	user, err := h.payoutUsecase.Link(r.Context(), userID, req.BankCode, req.BankName, req.AccountNumber)
	if err != nil {
		if errors.Is(err, usecase.ErrPhoneRequired) {
			respondError(w, http.StatusConflict, "phone_required", "add a phone number to your account before linking a payout account")
			return
		}
		respondError(w, http.StatusBadRequest, "payout_account_link_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "payout account linked successfully", user.PayoutAccount)
}
