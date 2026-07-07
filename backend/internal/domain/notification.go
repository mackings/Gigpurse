package domain

import (
	"context"
	"time"
)

type Notification struct {
	ID      string `json:"id" bson:"_id"`
	UserID  string `json:"user_id" bson:"user_id"`
	Title   string `json:"title" bson:"title"`
	Message string `json:"message" bson:"message"`
	// ContractID, when set, lets the frontend deep-link this notification to
	// that contract's chat thread (e.g. milestone proposed/accepted/funded).
	ContractID string `json:"contract_id,omitempty" bson:"contract_id,omitempty"`
	// Link, when set, is a ready-to-navigate frontend path — the general
	// mechanism for "clicking this notification takes you somewhere useful"
	// (a booking request, a contract, the disputes list, etc). Prefer this
	// over ContractID for new notification types; ContractID stays for the
	// existing chat-deep-link case.
	Link      string    `json:"link,omitempty" bson:"link,omitempty"`
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
