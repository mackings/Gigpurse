package domain

import (
	"context"
	"time"
)

type EmailVerificationToken struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	TokenHash string    `json:"-" bson:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
	UsedAt    time.Time `json:"used_at,omitempty" bson:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type EmailVerificationRepository interface {
	Create(ctx context.Context, token *EmailVerificationToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)
	MarkUsed(ctx context.Context, id string, usedAt time.Time) error
}
