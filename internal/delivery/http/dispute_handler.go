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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			ContractID string `json:"contract_id"`
			Reason     string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		dispute, err := h.disputeUsecase.OpenDispute(r.Context(), userID, req.ContractID, req.Reason)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dispute)
	case http.MethodGet:
		disputes, err := h.disputeUsecase.ListUserDisputes(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(disputes)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DisputeHandler) HandleAdminDisputes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "admin" {
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return
	}
	disputes, err := h.disputeUsecase.ListAllDisputes(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(disputes)
}

func (h *DisputeHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "admin" {
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return
	}
	var req struct {
		DisputeID  string `json:"dispute_id"`
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	dispute, err := h.disputeUsecase.ResolveDispute(r.Context(), req.DisputeID, req.Resolution)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dispute)
}
