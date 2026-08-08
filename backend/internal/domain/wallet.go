package domain

import (
	"context"
	"time"
)

// Wallet is a legacy internal record — Balance/EscrowBalance haven't been
// authoritative since money moved to PayPetal TrustCore (nothing mutates
// them anymore; every real transfer is either a one-off checkout or a
// direct bank payout, neither of which has a reloadable-balance concept).
// Kept only so WalletRepository's storage shape doesn't need a migration;
// WalletSummary below is what the API actually exposes.
type Wallet struct {
	UserID        string    `json:"-" bson:"user_id"`
	Balance       float64   `json:"-" bson:"balance"`
	EscrowBalance float64   `json:"-" bson:"escrow_balance"`
	TotalEarned   float64   `json:"-" bson:"total_earned"`
	TotalSpent    float64   `json:"-" bson:"total_spent"`
	UpdatedAt     time.Time `json:"-" bson:"updated_at"`
}

// WalletSummary is what GetWallet actually returns — TotalEarned/TotalSpent
// are computed by summing the Transaction log (the log is the source of
// truth now, not a mutated running balance), plus whether this user has a
// payout account on file, which is what the Wallet page needs to show
// instead of a spendable balance.
type WalletSummary struct {
	UserID           string         `json:"user_id"`
	TotalEarned      float64        `json:"total_earned"`
	TotalSpent       float64        `json:"total_spent"`
	HasPayoutAccount bool           `json:"has_payout_account"`
	PayoutAccount    *PayoutAccount `json:"payout_account,omitempty"`
}

type Transaction struct {
	ID          string    `json:"id" bson:"_id"`
	UserID      string    `json:"user_id" bson:"user_id"`
	Type        string    `json:"type" bson:"type"` // deposit, withdrawal, escrow_hold, escrow_release, payment_received
	Amount      float64   `json:"amount" bson:"amount"`
	Description string    `json:"description" bson:"description"`
	Reference   string    `json:"reference,omitempty" bson:"reference,omitempty"` // links back to EscrowAgreement.ReferenceID
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

type WalletRepository interface {
	GetOrCreate(ctx context.Context, userID string) (*Wallet, error)
	Save(ctx context.Context, wallet *Wallet) error
	AddTransaction(ctx context.Context, tx *Transaction) error
	ListTransactions(ctx context.Context, userID string) ([]*Transaction, error)
}

// EscrowHolding is one payment a client currently has held in escrow —
// either a whole job hire or a single funded milestone — enriched for
// display with what it's for and who the other party is. The wallet page
// sums these into "currently in escrow" and lists them individually, each
// with its own refund action.
type EscrowHolding struct {
	ReferenceID      string    `json:"reference_id"`
	ScopeType        string    `json:"scope_type"` // "job_hire" | "milestone"
	Title            string    `json:"title"`
	CounterpartyName string    `json:"counterparty_name"`
	Amount           float64   `json:"amount"`
	CreatedAt        time.Time `json:"created_at"`
}

type WalletUsecase interface {
	GetWallet(ctx context.Context, userID string) (*WalletSummary, error)
	ListTransactions(ctx context.Context, userID string) ([]*Transaction, error)

	// ListEscrowHoldings returns every payment this client currently has
	// held in escrow (paid in, not yet paid out or refunded) — job hires
	// and funded milestones alike.
	ListEscrowHoldings(ctx context.Context, userID string) ([]*EscrowHolding, error)
	// RequestRefund pulls a specific holding back to the client — routed to
	// jobUsecase.RequestHireRefund or milestoneUsecase.RequestRefund
	// depending on the holding's scope type.
	RequestRefund(ctx context.Context, userID, referenceID string) error
}
