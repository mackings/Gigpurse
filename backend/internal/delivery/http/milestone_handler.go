package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"gigpurse/internal/domain"
	"gigpurse/internal/usecase"
)

type MilestoneHandler struct {
	milestoneUsecase domain.MilestoneUsecase
	disputeUsecase   domain.DisputeUsecase
}

func NewMilestoneHandler(mu domain.MilestoneUsecase, du domain.DisputeUsecase) *MilestoneHandler {
	return &MilestoneHandler{milestoneUsecase: mu, disputeUsecase: du}
}

func (h *MilestoneHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/milestones", JWTMiddleware(h.HandleMilestones))
	mux.HandleFunc("/milestones/accept", JWTMiddleware(h.Accept))
	mux.HandleFunc("/milestones/reject", JWTMiddleware(h.Reject))
	mux.HandleFunc("/milestones/withdraw", JWTMiddleware(h.Withdraw))
	mux.HandleFunc("/milestones/counter", JWTMiddleware(h.Counter))
	mux.HandleFunc("/milestones/fund", JWTMiddleware(h.Fund))
	mux.HandleFunc("/milestones/fund/finalize", JWTMiddleware(h.FinalizeFund))
	mux.HandleFunc("/milestones/release", JWTMiddleware(h.Release))
	mux.HandleFunc("/milestones/cancel", JWTMiddleware(h.Cancel))
	mux.HandleFunc("/milestones/request-release", JWTMiddleware(h.RequestRelease))
}

func (h *MilestoneHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req milestoneActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	milestone, err := h.disputeUsecase.CancelMilestone(r.Context(), userID, req.ContractID, req.MilestoneID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "milestone_cancel_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "milestone cancelled successfully", milestone)
}

func (h *MilestoneHandler) RequestRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req milestoneActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.milestoneUsecase.RequestRelease(r.Context(), req.ContractID, req.MilestoneID, userID); err != nil {
		respondError(w, http.StatusBadRequest, "milestone_request_release_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "release requested successfully", map[string]string{"milestone_id": req.MilestoneID})
}

func (h *MilestoneHandler) HandleMilestones(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		contractID := r.URL.Query().Get("contract_id")
		if contractID == "" {
			respondError(w, http.StatusBadRequest, "missing_contract_id", "contract_id query parameter is required")
			return
		}
		milestones, err := h.milestoneUsecase.List(r.Context(), contractID, userID, role)
		if err != nil {
			respondError(w, http.StatusBadRequest, "milestones_list_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusOK, "milestones retrieved successfully", milestones)
	case http.MethodPost:
		var req struct {
			ContractID string                  `json:"contract_id"`
			Milestones []domain.MilestoneInput `json:"milestones"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		milestones, err := h.milestoneUsecase.Propose(r.Context(), req.ContractID, userID, req.Milestones)
		if err != nil {
			respondError(w, http.StatusBadRequest, "milestone_propose_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusCreated, "milestones proposed successfully", milestones)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

type milestoneActionRequest struct {
	ContractID  string `json:"contract_id"`
	MilestoneID string `json:"milestone_id"`
}

func (h *MilestoneHandler) Accept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req milestoneActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	milestone, err := h.milestoneUsecase.Accept(r.Context(), req.ContractID, req.MilestoneID, userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "milestone_accept_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "milestone accepted successfully", milestone)
}

func (h *MilestoneHandler) Reject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req milestoneActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	milestone, err := h.milestoneUsecase.Reject(r.Context(), req.ContractID, req.MilestoneID, userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "milestone_reject_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "milestone rejected successfully", milestone)
}

func (h *MilestoneHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req milestoneActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.milestoneUsecase.Withdraw(r.Context(), req.ContractID, req.MilestoneID, userID); err != nil {
		respondError(w, http.StatusBadRequest, "milestone_withdraw_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "milestone withdrawn successfully", map[string]string{"milestone_id": req.MilestoneID})
}

func (h *MilestoneHandler) Counter(w http.ResponseWriter, r *http.Request) {
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
		ContractID  string `json:"contract_id"`
		MilestoneID string `json:"milestone_id"`
		domain.MilestoneInput
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	milestone, err := h.milestoneUsecase.Counter(r.Context(), req.ContractID, req.MilestoneID, userID, req.MilestoneInput)
	if err != nil {
		respondError(w, http.StatusBadRequest, "milestone_counter_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "counter-offer saved successfully", milestone)
}

func (h *MilestoneHandler) Fund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req milestoneActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	paymentURL, reference, err := h.milestoneUsecase.Fund(r.Context(), req.ContractID, req.MilestoneID, userID)
	if err != nil {
		if errors.Is(err, usecase.ErrPayoutAccountRequired) {
			respondError(w, http.StatusConflict, "payout_account_required", "this musician hasn't set up payouts yet — they've been notified to add a bank account")
			return
		}
		if errors.Is(err, usecase.ErrPhoneRequired) {
			respondError(w, http.StatusConflict, "phone_required", "add a phone number to your account before funding this milestone")
			return
		}
		respondError(w, http.StatusBadRequest, "milestone_fund_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "payment started", map[string]string{
		"payment_url": paymentURL,
		"reference":   reference,
	})
}

// FinalizeFund is polled by the frontend after the client returns from
// PayPetal's hosted checkout (the webhook also calls the same usecase
// method independently — whichever lands first wins, both are idempotent).
func (h *MilestoneHandler) FinalizeFund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if _, _, ok := GetUserFromContext(r.Context()); !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	milestone, err := h.milestoneUsecase.FinalizeFund(r.Context(), req.Reference)
	if err != nil {
		respondError(w, http.StatusBadRequest, "milestone_finalize_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "milestone funded successfully", milestone)
}

func (h *MilestoneHandler) Release(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req milestoneActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	milestone, err := h.milestoneUsecase.Release(r.Context(), req.ContractID, req.MilestoneID, userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "milestone_release_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "milestone released successfully", milestone)
}
