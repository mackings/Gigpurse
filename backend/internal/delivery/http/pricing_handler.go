package http

import (
	"net/http"

	"gigpurse/internal/usecase"
)

// PricingHandler exposes the platform's fee structure — a single source of
// truth so the frontend's Pricing page and fee breakdowns aren't hardcoding
// numbers that could drift from what's actually charged (see usecase/pricing.go).
type PricingHandler struct{}

func NewPricingHandler() *PricingHandler {
	return &PricingHandler{}
}

func (h *PricingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/pricing", h.Get)
}

func (h *PricingHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	respondSuccess(w, http.StatusOK, "pricing retrieved successfully", map[string]any{
		"talent_commission_rate":  usecase.TalentCommissionRate,
		"client_service_fee_rate": usecase.ClientServiceFeeRate,
	})
}
