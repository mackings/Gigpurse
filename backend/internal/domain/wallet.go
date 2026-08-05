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

type WalletUsecase interface {
	GetWallet(ctx context.Context, userID string) (*WalletSummary, error)
	ListTransactions(ctx context.Context, userID string) ([]*Transaction, error)
}
