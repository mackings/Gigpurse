package http

import (
	"encoding/json"
	"net/http"

	"gigpurse/internal/domain"
)

// PayoutHandler is the admin queue of money GigPurse owes after TrustVault
// releases — see domain.Payout.
type PayoutHandler struct {
	payoutUsecase domain.PayoutUsecase
}

func NewPayoutHandler(pu domain.PayoutUsecase) *PayoutHandler {
	return &PayoutHandler{payoutUsecase: pu}
}

func (h *PayoutHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/payouts", JWTMiddleware(h.List))
	mux.HandleFunc("POST /admin/payouts/mark-paid", JWTMiddleware(h.MarkPaid))
}

func (h *PayoutHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, role, ok := GetUserFromContext(r.Context()); !ok || role != "admin" {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin role required")
		return
	}
	payouts, err := h.payoutUsecase.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "payouts_list_failed", err.Error())
		return
	}
	if payouts == nil {
		payouts = []*domain.Payout{}
	}
	respondSuccess(w, http.StatusOK, "payouts retrieved successfully", payouts)
}

func (h *PayoutHandler) MarkPaid(w http.ResponseWriter, r *http.Request) {
	adminID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "admin" {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin role required")
		return
	}
	var req struct {
		ID   string `json:"id"`
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	payout, err := h.payoutUsecase.MarkPaid(r.Context(), adminID, req.ID, req.Note)
	if err != nil {
		respondError(w, http.StatusBadRequest, "payout_mark_paid_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "payout marked paid", payout)
}
