package http

import (
	"encoding/json"
	"net/http"

	"gigpurse/internal/domain"
)

type DisputeHandler struct {
	disputeUsecase domain.DisputeUsecase
}

func NewDisputeHandler(du domain.DisputeUsecase) *DisputeHandler {
	return &DisputeHandler{disputeUsecase: du}
}

func (h *DisputeHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/disputes", JWTMiddleware(h.HandleUserDisputes))
	mux.HandleFunc("/admin/disputes", JWTMiddleware(h.HandleAdminDisputes))
	mux.HandleFunc("/admin/disputes/resolve", JWTMiddleware(h.ResolveDispute))
}

func (h *DisputeHandler) HandleUserDisputes(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			ContractID string `json:"contract_id"`
			Reason     string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		dispute, err := h.disputeUsecase.OpenDispute(r.Context(), userID, req.ContractID, req.Reason)
		if err != nil {
			respondError(w, http.StatusBadRequest, "dispute_open_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusCreated, "dispute opened successfully", dispute)
	case http.MethodGet:
		disputes, err := h.disputeUsecase.ListUserDisputes(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "disputes_list_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusOK, "disputes retrieved successfully", disputes)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *DisputeHandler) HandleAdminDisputes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	_, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "admin" {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin role required")
		return
	}
	disputes, err := h.disputeUsecase.ListAllDisputes(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "disputes_list_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "admin disputes retrieved successfully", disputes)
}

func (h *DisputeHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	_, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "admin" {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin role required")
		return
	}
	var req struct {
		DisputeID  string `json:"dispute_id"`
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	dispute, err := h.disputeUsecase.ResolveDispute(r.Context(), req.DisputeID, req.Resolution)
	if err != nil {
		respondError(w, http.StatusBadRequest, "dispute_resolve_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "dispute resolved successfully", dispute)
}
