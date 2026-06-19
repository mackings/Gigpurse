package http

import (
	"encoding/json"
	"net/http"

	"gigpurse/internal/domain"
)

type NotificationHandler struct {
	notifUsecase domain.NotificationUsecase
}

func NewNotificationHandler(nu domain.NotificationUsecase) *NotificationHandler {
	return &NotificationHandler{
		notifUsecase: nu,
	}
}

func (h *NotificationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/notifications", JWTMiddleware(h.HandleNotifications))
	mux.HandleFunc("/notifications/read", JWTMiddleware(h.MarkAsRead))
}

func (h *NotificationHandler) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	notifications, err := h.notifUsecase.ListForUser(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "notifications_list_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "notifications retrieved successfully", notifications)
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
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
		NotificationID string `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	err := h.notifUsecase.MarkAsRead(r.Context(), req.NotificationID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "notification_mark_read_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "notification marked as read", map[string]string{
		"message": "notification marked as read",
	})
}
