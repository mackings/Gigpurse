package domain

import (
	"context"
	"time"
)

// PushSubscription is one browser/device's Web Push registration for a
// user. A user can have several (one per browser/device they've enabled
// push on); Endpoint uniquely identifies each one.
type PushSubscription struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Endpoint  string    `json:"endpoint" bson:"endpoint"`
	P256dh    string    `json:"p256dh" bson:"p256dh"`
	Auth      string    `json:"auth" bson:"auth"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type PushSubscriptionRepository interface {
	Upsert(ctx context.Context, sub *PushSubscription) error
	ListByUser(ctx context.Context, userID string) ([]*PushSubscription, error)
	DeleteByEndpoint(ctx context.Context, userID, endpoint string) error
}
