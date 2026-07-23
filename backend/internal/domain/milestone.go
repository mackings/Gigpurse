package domain

import (
	"context"
	"time"
)

// Milestone status lifecycle:
//
//	proposed  -- the client proposes a milestone for a contract; the talent
//	             can accept, reject, or counter it (a counter keeps the
//	             status at "proposed" but flips who's offering what — from
//	             then on it's whoever didn't make the last offer who can
//	             respond, so a countered proposal bounces back to the client)
//	accepted  -- the other party accepted the current terms, making it fundable
//	rejected  -- the other party rejects it (terminal)
//	funded    -- the client funds escrow for an accepted milestone
//	released  -- the client releases escrow, crediting the musician's wallet (terminal)
//	refunded  -- a dispute resolved in the client's favor, so held escrow
//	             went back to their wallet balance instead of the musician's (terminal)
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

	// LastReminderAt tracks the last time the awaiting-response party was
	// re-notified about this still-`proposed` milestone — nil until the
	// first reminder fires. See MilestoneUsecase reminder scanner.
	LastReminderAt *time.Time `json:"last_reminder_at,omitempty" bson:"last_reminder_at,omitempty"`
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
	Delete(ctx context.Context, id string) error
	// ListByStatus lists across every contract — used by the reminder
	// scanner to find every still-`proposed` milestone system-wide.
	ListByStatus(ctx context.Context, status string) ([]*Milestone, error)
}

type MilestoneUsecase interface {
	Propose(ctx context.Context, contractID, proposerID string, items []MilestoneInput) ([]*Milestone, error)
	Accept(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Reject(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Counter(ctx context.Context, contractID, milestoneID, userID string, terms MilestoneInput) (*Milestone, error)
	// Withdraw lets the proposer retract their own still-pending proposal
	// (e.g. they mistyped an amount or date) so they can send a corrected
	// one — only while it's awaiting a response, before it clutters the
	// history with a rejected/superseded entry.
	Withdraw(ctx context.Context, contractID, milestoneID, userID string) error
	Fund(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Release(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	List(ctx context.Context, contractID, requesterID string) ([]*Milestone, error)

	// RefundHeldForContract sweeps every still-`funded` milestone on a
	// contract back to the client's wallet balance — used when a dispute
	// resolves in the client's favor. Unlike Release/Fund this isn't gated
	// by a caller userID: the caller (dispute resolution) has already
	// established the resolver is a moderator/admin, so this is meant to be
	// invoked internally rather than exposed as its own end-user action.
	RefundHeldForContract(ctx context.Context, contractID string) error

	// StartReminderScanner runs in the background for the lifetime of ctx,
	// periodically re-notifying whoever hasn't responded to a still-`proposed`
	// milestone. Called once at startup from main.go.
	StartReminderScanner(ctx context.Context, checkInterval, nudgeAfter time.Duration)
}
