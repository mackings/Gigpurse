package domain

import (
	"context"
	"time"
)

// Milestone status lifecycle:
//
//	proposed  -- either party proposes a milestone for a contract; the other
//	             party can accept, reject, or counter it (a counter keeps the
//	             status at "proposed" but flips who's offering what)
//	accepted  -- the other party accepted the current terms, making it fundable
//	rejected  -- the other party rejects it (terminal)
//	funded    -- the client funds escrow for an accepted milestone
//	released  -- the client releases escrow, crediting the musician's wallet (terminal)
type Milestone struct {
	ID         string                      `json:"id" bson:"_id"`
	ContractID string                      `json:"contract_id" bson:"contract_id"`
	Title      string                      `json:"title" bson:"title"`
	Amount     float64                     `json:"amount" bson:"amount"`
	DueDate    *time.Time                  `json:"due_date,omitempty" bson:"due_date,omitempty"`
	Status     string                      `json:"status" bson:"status"`
	ProposedBy string                      `json:"proposed_by" bson:"proposed_by"`
	History    []MilestoneNegotiationEntry `json:"history,omitempty" bson:"history,omitempty"`
	Order      int                         `json:"order" bson:"order"`
	CreatedAt  time.Time                   `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time                   `json:"updated_at" bson:"updated_at"`
}

// MilestoneNegotiationEntry records one offer in a milestone's back-and-forth
// — every propose/counter appends one, so both parties can see who offered
// what terms, mirroring how direct-hire booking negotiation works.
type MilestoneNegotiationEntry struct {
	ProposedBy string     `json:"proposed_by" bson:"proposed_by"`
	Title      string     `json:"title,omitempty" bson:"title,omitempty"`
	Amount     float64    `json:"amount" bson:"amount"`
	DueDate    *time.Time `json:"due_date,omitempty" bson:"due_date,omitempty"`
	CreatedAt  time.Time  `json:"created_at" bson:"created_at"`
}

type MilestoneInput struct {
	Title   string     `json:"title"`
	Amount  float64    `json:"amount"`
	DueDate *time.Time `json:"due_date,omitempty"`
}

type MilestoneRepository interface {
	Create(ctx context.Context, m *Milestone) error
	GetByID(ctx context.Context, id string) (*Milestone, error)
	ListByContract(ctx context.Context, contractID string) ([]*Milestone, error)
	Update(ctx context.Context, m *Milestone) error
}

type MilestoneUsecase interface {
	Propose(ctx context.Context, contractID, proposerID string, items []MilestoneInput) ([]*Milestone, error)
	Accept(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Reject(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Counter(ctx context.Context, contractID, milestoneID, userID string, terms MilestoneInput) (*Milestone, error)
	Fund(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Release(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	List(ctx context.Context, contractID, requesterID string) ([]*Milestone, error)
}
