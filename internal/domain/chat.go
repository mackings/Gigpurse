package domain

import (
	"context"
	"time"
)

type ChatMessage struct {
	ID        string    `json:"id" bson:"_id"`
	SenderID  string    `json:"sender_id" bson:"sender_id"`
	RecvID    string    `json:"recv_id" bson:"recv_id"`
	Content   string    `json:"content" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

type ChatRepository interface {
	SaveMessage(ctx context.Context, msg *ChatMessage) error
	GetChatHistory(ctx context.Context, user1, user2 string) ([]*ChatMessage, error)
	GetRecentChats(ctx context.Context, userID string) ([]*ChatMessage, error)
}

type ChatUsecase interface {
	SendMessage(ctx context.Context, senderID, recvID, content string) (*ChatMessage, error)
	GetChatHistory(ctx context.Context, user1, user2 string) ([]*ChatMessage, error)
	GetRecentChats(ctx context.Context, userID string) ([]*ChatMessage, error)
}
