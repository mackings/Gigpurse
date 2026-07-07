package http

import (
	"encoding/json"
	"net/http"
	"time"

	"gigpurse/internal/domain"
)

type PushHandler struct {
	subRepo  domain.PushSubscriptionRepository
	vapidPub string
}

func NewPushHandler(subRepo domain.PushSubscriptionRepository, vapidPub string) *PushHandler {
	return &PushHandler{subRepo: subRepo, vapidPub: vapidPub}
}

func (h *PushHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/push/vapid-public-key", h.VAPIDPublicKey)
	mux.HandleFunc("/push/subscribe", JWTMiddleware(h.Subscribe))
	mux.HandleFunc("/push/unsubscribe", JWTMiddleware(h.Unsubscribe))
}

// VAPIDPublicKey is unauthenticated — the browser needs it before the user
// necessarily has a session (it's the applicationServerKey passed to
// pushManager.subscribe).
func (h *PushHandler) VAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	respondSuccess(w, http.StatusOK, "vapid public key retrieved successfully", map[string]string{"public_key": h.vapidPub})
}

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req pushSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "endpoint and keys.p256dh/keys.auth are required")
		return
	}
	sub := &domain.PushSubscription{
		UserID:    userID,
		Endpoint:  req.Endpoint,
		P256dh:    req.Keys.P256dh,
		Auth:      req.Keys.Auth,
		CreatedAt: time.Now(),
	}
	if err := h.subRepo.Upsert(r.Context(), sub); err != nil {
		respondError(w, http.StatusInternalServerError, "push_subscribe_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusCreated, "push subscription saved successfully", sub)
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
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
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Endpoint == "" {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "endpoint is required")
		return
	}
	if err := h.subRepo.DeleteByEndpoint(r.Context(), userID, req.Endpoint); err != nil {
		respondError(w, http.StatusInternalServerError, "push_unsubscribe_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "push subscription removed successfully", nil)
}
