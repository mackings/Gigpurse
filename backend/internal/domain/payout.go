package domain

import (
	"context"
	"strings"
	"time"
)

// Payout is money GigPurse owes someone after a TrustVault release, which
// lands in GigPurse's own PayPetal wallet. When a payout sub-wallet is set
// up, GigPurse pays it automatically (a TrustCore agreement the sub-wallet
// funds, released to the payee's bank — see usecase/auto_payout.go);
// otherwise, or if an admin takes it over, an admin pays it by hand. Exactly
// one row exists per (agreement, kind) — ID is built from the two, so
// recording the same payout twice is impossible.
//
// Status: pending (release asked for, PayPetal not yet confirmed) -> due
// (money is in GigPurse's wallet, owed) -> sending (PayPetal is paying it
// automatically) -> paid. An admin stopping an automatic payout moves it to
// stopping until PayPetal has returned the money to the sub-wallet, then back
// to due for paying by hand.
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
	RevertTo string `json:"-" bson:"revert_to"`

	// Automatic payout. AutoRef is the PayPetal TrustCore reference, fixed
	// per payout so a retry can never fund a second transfer.
	AutoRef      string `json:"auto_ref,omitempty" bson:"auto_ref,omitempty"`
	AutoError    string `json:"auto_error,omitempty" bson:"auto_error,omitempty"`       // what it's waiting on, for the admin
	AutoAttempts int    `json:"auto_attempts,omitempty" bson:"auto_attempts,omitempty"` // failures in a row; sets the retry gap
	// AutoRound moves to a fresh AutoRef, only once PayPetal has handed the
	// last one's money back.
	AutoRound  int        `json:"-" bson:"auto_round,omitempty"`
	RetryAt    *time.Time `json:"retry_at,omitempty" bson:"retry_at,omitempty"`
	ManualOnly bool       `json:"manual_only,omitempty" bson:"manual_only,omitempty"` // an admin is paying this by hand
	// WaitingOn is what the next automatic try waits for, besides RetryAt:
	// "funds" (a top-up of the payout wallet), "account" (the payee fixing
	// their bank account), "transfer" (retrying a failed bank transfer), or
	// "" (a plain retry). The system retries on its own, indefinitely.
	WaitingOn   string    `json:"waiting_on,omitempty" bson:"waiting_on,omitempty"`
	BalanceSeen float64   `json:"-" bson:"balance_seen,omitempty"`
	AccountSeen time.Time `json:"-" bson:"account_seen,omitempty"`
	// Held: PayPetal's record didn't match this payout, so nothing more
	// happens to it automatically until an admin has looked.
	Held bool `json:"held,omitempty" bson:"held,omitempty"`

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
	// UpdateIfStatus saves p only if the stored row is still in status, so a
	// stale copy can't overwrite a change made meanwhile.
	UpdateIfStatus(ctx context.Context, p *Payout, status string) (bool, error)
}

// AutoPayoutInfo is the payout sub-wallet, for the admin page: whether
// automatic payouts are on, and how much is left to pay with.
type AutoPayoutInfo struct {
	Enabled       bool    `json:"enabled"`
	Reference     string  `json:"reference,omitempty"`
	Balance       float64 `json:"balance"`
	BankName      string  `json:"bank_name,omitempty"`
	AccountNumber string  `json:"account_number,omitempty"`
	AccountName   string  `json:"account_name,omitempty"`
	Error         string  `json:"error,omitempty"`
}

type PayoutUsecase interface {
	List(ctx context.Context, status string) ([]*Payout, error)
	// MarkPaid records that an admin has sent the money by bank transfer.
	MarkPaid(ctx context.Context, adminID, payoutID, note string) (*Payout, error)

	// AutoPayouts reports on the payout sub-wallet.
	AutoPayouts(ctx context.Context) (*AutoPayoutInfo, error)
	// ProcessAuto moves every automatic payout one step on. Safe to call
	// often and concurrently.
	ProcessAuto(ctx context.Context)
	// RetryAuto tries a payout again now: one that's held, taken over by an
	// admin, or just waiting. Never needed for a payout to go out.
	RetryAuto(ctx context.Context, payoutID string) (*Payout, error)
	// PayByHand takes a payout off automatic payment, getting back any money
	// already set aside for it, so an admin can pay it themselves.
	PayByHand(ctx context.Context, payoutID string) (*Payout, error)
}

// IsAutoPayoutReference reports whether a PayPetal reference is one of
// GigPurse's automatic payouts: "pt" (talent) or "pr" (refund) plus the
// milestone's "ms…" payment reference.
func IsAutoPayoutReference(ref string) bool {
	return strings.HasPrefix(ref, "ptms") || strings.HasPrefix(ref, "prms")
}
