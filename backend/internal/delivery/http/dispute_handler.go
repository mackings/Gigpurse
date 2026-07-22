package http

import (
	"encoding/json"
	"net/http"

	"gigpurse/internal/domain"
)

type DisputeHandler struct {
	disputeUsecase domain.DisputeUsecase
	hub            *Hub
}

func NewDisputeHandler(du domain.DisputeUsecase, hub *Hub) *DisputeHandler {
	return &DisputeHandler{disputeUsecase: du, hub: hub}
}

func (h *DisputeHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/disputes", JWTMiddleware(h.HandleUserDisputes))
	mux.HandleFunc("/disputes/join", JWTMiddleware(h.JoinDispute))
	mux.HandleFunc("/disputes/messages", JWTMiddleware(h.HandleDisputeMessages))
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
		if id := r.URL.Query().Get("id"); id != "" {
			dispute, err := h.disputeUsecase.GetDispute(r.Context(), userID, id)
			if err != nil {
				respondError(w, http.StatusNotFound, "dispute_not_found", err.Error())
				return
			}
			respondSuccess(w, http.StatusOK, "dispute retrieved successfully", dispute)
			return
		}
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
	if !ok || (role != "admin" && role != "moderator") {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin or moderator role required")
		return
	}
	disputes, err := h.disputeUsecase.ListAllDisputes(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "disputes_list_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "admin disputes retrieved successfully", disputes)
}

func (h *DisputeHandler) JoinDispute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || (role != "admin" && role != "moderator") {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin or moderator role required")
		return
	}
	var req struct {
		DisputeID string `json:"dispute_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	dispute, err := h.disputeUsecase.JoinDispute(r.Context(), userID, req.DisputeID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "dispute_join_failed", err.Error())
		return
	}
	// The join system-message was already persisted by the usecase — fan out
	// a light "refresh" ping so anyone with the room open picks it up live
	// without needing a full message payload here (the messages list
	// endpoint is the source of truth for content).
	h.hub.SendToMany([]string{dispute.ClientID, dispute.MusicianID, dispute.ModeratorID}, "dispute_updated", dispute)
	respondSuccess(w, http.StatusOK, "joined dispute successfully", dispute)
}

func (h *DisputeHandler) HandleDisputeMessages(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		disputeID := r.URL.Query().Get("dispute_id")
		if disputeID == "" {
			respondError(w, http.StatusBadRequest, "missing_dispute_id", "dispute_id query parameter is required")
			return
		}
		messages, err := h.disputeUsecase.ListDisputeMessages(r.Context(), userID, disputeID)
		if err != nil {
			respondError(w, http.StatusForbidden, "dispute_messages_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusOK, "dispute messages retrieved successfully", messages)
	case http.MethodPost:
		var req struct {
			DisputeID       string `json:"dispute_id"`
			Content         string `json:"content"`
			AttachmentURL   string `json:"attachment_url"`
			AttachmentType  string `json:"attachment_type"`
			MentionedUserID string `json:"mentioned_user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		msg, err := h.disputeUsecase.SendDisputeMessage(r.Context(), userID, req.DisputeID, req.Content, req.AttachmentURL, req.AttachmentType, req.MentionedUserID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "dispute_message_failed", err.Error())
			return
		}
		if dispute, err := h.disputeUsecase.GetDispute(r.Context(), userID, req.DisputeID); err == nil {
			h.hub.SendToMany([]string{dispute.ClientID, dispute.MusicianID, dispute.ModeratorID}, "dispute_message", msg)
		}
		respondSuccess(w, http.StatusCreated, "dispute message sent successfully", msg)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *DisputeHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || (role != "admin" && role != "moderator") {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin or moderator role required")
		return
	}
	var req struct {
		DisputeID  string `json:"dispute_id"`
		WinnerID   string `json:"winner_id"`
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	dispute, err := h.disputeUsecase.ResolveDispute(r.Context(), userID, req.DisputeID, req.WinnerID, req.Resolution)
	if err != nil {
		respondError(w, http.StatusBadRequest, "dispute_resolve_failed", err.Error())
		return
	}
	h.hub.SendToMany([]string{dispute.ClientID, dispute.MusicianID, dispute.ModeratorID}, "dispute_updated", dispute)
	respondSuccess(w, http.StatusOK, "dispute resolved successfully", dispute)
}
