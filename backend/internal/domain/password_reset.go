package domain

import (
	"context"
	"time"
)

type PasswordResetToken struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	TokenHash string    `json:"-" bson:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
	UsedAt    time.Time `json:"used_at,omitempty" bson:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type PasswordResetRepository interface {
	Create(ctx context.Context, token *PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string, usedAt time.Time) error
}
