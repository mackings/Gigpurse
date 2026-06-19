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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	notifications, err := h.notifUsecase.ListForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		NotificationID string `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.notifUsecase.MarkAsRead(r.Context(), req.NotificationID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "notification marked as read",
	})
}
