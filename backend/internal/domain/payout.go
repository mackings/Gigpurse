package domain

import (
	"context"
	"time"
)

// Payout is money GigPurse owes someone after a TrustVault release.
// TrustVault releases land in GigPurse's own PayPetal wallet and PayPetal has
// no API to send money on from there, so an admin pays each of these by hand
// and records it. Exactly one row exists per (agreement, kind) — ID is built
// from the two, so recording the same payout twice is impossible.
//
// Status: pending (release asked for, PayPetal not yet confirmed) -> due
// (money is in GigPurse's wallet, owed) -> paid.
type Payout struct {
	ID          string  `json:"id" bson:"_id"`
	ReferenceID string  `json:"reference_id" bson:"reference_id"`
	MilestoneID string  `json:"milestone_id" bson:"milestone_id"`
	ContractID  string  `json:"contract_id" bson:"contract_id"`
	UserID      string  `json:"user_id" bson:"user_id"`
	Kind        string  `json:"kind" bson:"kind"` // "talent_payout" | "client_refund"
	Title       string  `json:"title" bson:"title"`
	Amount      float64 `json:"amount" bson:"amount"`
	Status      string  `json:"status" bson:"status"`
	// RevertTo is the milestone status to go back to if PayPetal turns out
	// not to have released the money after all.
	RevertTo  string     `json:"-" bson:"revert_to"`
	Note      string     `json:"note,omitempty" bson:"note,omitempty"` // bank transfer reference
	PaidBy    string     `json:"paid_by,omitempty" bson:"paid_by,omitempty"`
	PaidAt    *time.Time `json:"paid_at,omitempty" bson:"paid_at,omitempty"`
	CreatedAt time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" bson:"updated_at"`

	// Filled in when listed for an admin, never persisted.
	UserName string         `json:"user_name,omitempty" bson:"-"`
	Account  *PayoutAccount `json:"account,omitempty" bson:"-"`
}

const (
	PayoutTalent = "talent_payout"
	PayoutRefund = "client_refund"
)

type PayoutRepository interface {
	// ReplacePending swaps in a fresh set of pending rows for an agreement.
	// Rows already due or paid are never touched.
	ReplacePending(ctx context.Context, referenceID string, payouts []*Payout) error
	GetByID(ctx context.Context, id string) (*Payout, error)
	ListByStatus(ctx context.Context, status string) ([]*Payout, error)
	// CompareAndSwapStatus reports whether this call made the change — the
	// guard against paying or announcing the same payout twice.
	CompareAndSwapStatus(ctx context.Context, id, from, to string) (bool, error)
	Update(ctx context.Context, p *Payout) error
}

type PayoutUsecase interface {
	List(ctx context.Context, status string) ([]*Payout, error)
	// MarkPaid records that an admin has sent the money by bank transfer.
	MarkPaid(ctx context.Context, adminID, payoutID, note string) (*Payout, error)
}
