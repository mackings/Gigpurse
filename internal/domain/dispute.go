package domain

import (
	"context"
	"time"
)

type Dispute struct {
	ID         string    `json:"id" bson:"_id"`
	ContractID string    `json:"contract_id" bson:"contract_id"`
	ClientID   string    `json:"client_id" bson:"client_id"`
	MusicianID string    `json:"musician_id" bson:"musician_id"`
	OpenedByID string    `json:"opened_by_id" bson:"opened_by_id"`
	Reason     string    `json:"reason" bson:"reason"`
	Status     string    `json:"status" bson:"status"` // "open", "resolved", "closed"
	Resolution string    `json:"resolution,omitempty" bson:"resolution,omitempty"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" bson:"updated_at"`
}

type DisputeRepository interface {
	Create(ctx context.Context, dispute *Dispute) error
	GetByID(ctx context.Context, id string) (*Dispute, error)
	Update(ctx context.Context, dispute *Dispute) error
	List(ctx context.Context, status string) ([]*Dispute, error)
	ListForUser(ctx context.Context, userID string) ([]*Dispute, error)
}

type DisputeUsecase interface {
	OpenDispute(ctx context.Context, userID, contractID, reason string) (*Dispute, error)
	ListUserDisputes(ctx context.Context, userID string) ([]*Dispute, error)
	ListAllDisputes(ctx context.Context, status string) ([]*Dispute, error)
	ResolveDispute(ctx context.Context, disputeID, resolution string) (*Dispute, error)
}
