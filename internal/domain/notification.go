package domain

import (
	"context"
	"time"
)

type Notification struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Title     string    `json:"title" bson:"title"`
	Message   string    `json:"message" bson:"message"`
	IsRead    bool      `json:"is_read" bson:"is_read"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type NotificationRepository interface {
	Create(ctx context.Context, notif *Notification) error
	ListForUser(ctx context.Context, userID string) ([]*Notification, error)
	MarkAsRead(ctx context.Context, notifID, userID string) error
}

type NotificationUsecase interface {
	Create(ctx context.Context, userID, title, message string) (*Notification, error)
	ListForUser(ctx context.Context, userID string) ([]*Notification, error)
	MarkAsRead(ctx context.Context, notifID, userID string) error
}
