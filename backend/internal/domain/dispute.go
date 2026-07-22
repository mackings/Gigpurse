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

	// ModeratorID is empty until a moderator/admin joins the dispute's chat
	// room — the two original parties can't message each other in that room
	// until this is set (see DisputeChatUsecase.SendMessage).
	ModeratorID string `json:"moderator_id,omitempty" bson:"moderator_id,omitempty"`

	Resolution string `json:"resolution,omitempty" bson:"resolution,omitempty"`
	// WinnerID is the ClientID or MusicianID of whoever the moderator ruled
	// in favor of — resolving requires picking exactly one, since a dispute
	// is adversarial by nature (this also decides which side any held
	// escrow moves to on resolution).
	WinnerID string `json:"winner_id,omitempty" bson:"winner_id,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
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
	GetDispute(ctx context.Context, requesterID, disputeID string) (*Dispute, error)
	// ResolveDispute requires a winnerID (the dispute's ClientID or
	// MusicianID) — resolving also sweeps any escrow still held against the
	// dispute's contract (funded milestones, and job-level escrow for a
	// job-sourced contract with no milestones) to the winner, or back to the
	// client if the client won.
	ResolveDispute(ctx context.Context, resolverID, disputeID, winnerID, resolution string) (*Dispute, error)

	// JoinDispute lets a moderator/admin attach themselves to a dispute's
	// chat room — this is what unblocks messaging between the two original
	// parties, and posts an automatic system message announcing it.
	JoinDispute(ctx context.Context, moderatorID, disputeID string) (*Dispute, error)
	SendDisputeMessage(ctx context.Context, senderID, disputeID, content, attachmentURL, attachmentType, mentionedUserID string) (*ChatMessage, error)
	ListDisputeMessages(ctx context.Context, requesterID, disputeID string) ([]*ChatMessage, error)
}
