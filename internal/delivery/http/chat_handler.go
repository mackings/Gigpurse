package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gigpurse/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	chatUsecase domain.ChatUsecase
	// WebSocket Connection Hub
	upgrader websocket.Upgrader
	mu       sync.RWMutex
	clients  map[string]*websocket.Conn
}

func NewChatHandler(cu domain.ChatUsecase) *ChatHandler {
	return &ChatHandler{
		chatUsecase: cu,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for the API gateway/Render hosting
				return true
			},
		},
		clients: make(map[string]*websocket.Conn),
	}
}

func (h *ChatHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/chats", JWTMiddleware(h.HandleChats))
	mux.HandleFunc("/chats/history", JWTMiddleware(h.GetChatHistory))
	mux.HandleFunc("/chats/recent", JWTMiddleware(h.GetRecentChats))
	mux.HandleFunc("/chats/ws", h.HandleWebSocket)
}

func (h *ChatHandler) HandleChats(w http.ResponseWriter, r *http.Request) {
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
		RecvID  string `json:"recv_id"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	msg, err := h.chatUsecase.SendMessage(r.Context(), userID, req.RecvID, req.Content)
	if err != nil {
		respondError(w, http.StatusBadRequest, "chat_send_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusCreated, "message sent successfully", msg)
}

func (h *ChatHandler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	otherUserID := r.URL.Query().Get("user_id")
	if otherUserID == "" {
		respondError(w, http.StatusBadRequest, "missing_user_id", "user_id query parameter is required")
		return
	}

	history, err := h.chatUsecase.GetChatHistory(r.Context(), userID, otherUserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "chat_history_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "chat history retrieved successfully", history)
}

func (h *ChatHandler) GetRecentChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	recent, err := h.chatUsecase.GetRecentChats(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "recent_chats_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "recent chats retrieved successfully", recent)
}

// HandleWebSocket upgrades HTTP to WebSockets and handles real-time messages
func (h *ChatHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate the user from query param token (standard for websockets)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// Fallback to Header if available
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		respondError(w, http.StatusUnauthorized, "token_required", "unauthorized: token required")
		return
	}

	// Parse & Validate JWT Token
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		respondError(w, http.StatusUnauthorized, "invalid_token", "unauthorized: invalid token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		respondError(w, http.StatusUnauthorized, "invalid_token_claims", "unauthorized: invalid claims")
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "invalid_token_claims", "unauthorized: invalid user_id in token")
		return
	}

	// Upgrade the connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection for user %s: %v", userID, err)
		return
	}

	h.registerConnection(userID, conn)
	defer h.deregisterConnection(userID, conn)

	log.Printf("user %s connected via WebSockets", userID)

	// WebSocket Message Read Loop
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("user %s disconnected: %v", userID, err)
			break
		}

		var wsReq struct {
			RecvID  string `json:"recv_id"`
			Content string `json:"content"`
		}

		if err := json.Unmarshal(messageBytes, &wsReq); err != nil {
			_ = conn.WriteJSON(map[string]string{"error": "invalid message format"})
			continue
		}

		// Persist & Filter message using usecase
		// WS uses background context as base
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		msg, err := h.chatUsecase.SendMessage(ctx, userID, wsReq.RecvID, wsReq.Content)
		cancel()

		if err != nil {
			_ = conn.WriteJSON(map[string]string{"error": err.Error()})
			continue
		}

		// Send confirmation back to sender
		_ = conn.WriteJSON(msg)

		// Dispatch to receiver if online
		h.dispatchToUser(wsReq.RecvID, msg)
	}
}

func (h *ChatHandler) registerConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Clean up any old connection for the same user if it existed
	if oldConn, exists := h.clients[userID]; exists {
		_ = oldConn.Close()
	}
	h.clients[userID] = conn
}

func (h *ChatHandler) deregisterConnection(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if activeConn, exists := h.clients[userID]; exists && activeConn == conn {
		delete(h.clients, userID)
		_ = conn.Close()
	}
}

func (h *ChatHandler) dispatchToUser(userID string, msg *domain.ChatMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conn, online := h.clients[userID]; online {
		go func(c *websocket.Conn, m *domain.ChatMessage) {
			_ = c.WriteJSON(m)
		}(conn, msg)
	}
}
