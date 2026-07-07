package domain

import (
	"context"
	"time"
)

type Wallet struct {
	UserID        string    `json:"user_id" bson:"user_id"`
	Balance       float64   `json:"balance" bson:"balance"`
	EscrowBalance float64   `json:"escrow_balance" bson:"escrow_balance"`
	TotalEarned   float64   `json:"total_earned" bson:"total_earned"`
	TotalSpent    float64   `json:"total_spent" bson:"total_spent"`
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`
}

type Transaction struct {
	ID          string    `json:"id" bson:"_id"`
	UserID      string    `json:"user_id" bson:"user_id"`
	Type        string    `json:"type" bson:"type"` // deposit, withdrawal, escrow_hold, escrow_release, payment_received
	Amount      float64   `json:"amount" bson:"amount"`
	Description string    `json:"description" bson:"description"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

type WalletRepository interface {
	GetOrCreate(ctx context.Context, userID string) (*Wallet, error)
	Save(ctx context.Context, wallet *Wallet) error
	AddTransaction(ctx context.Context, tx *Transaction) error
	ListTransactions(ctx context.Context, userID string) ([]*Transaction, error)
}

type WalletUsecase interface {
	GetWallet(ctx context.Context, userID string) (*Wallet, error)
	ListTransactions(ctx context.Context, userID string) ([]*Transaction, error)
	Deposit(ctx context.Context, userID string, amount float64) (*Wallet, error)
	Withdraw(ctx context.Context, userID string, amount float64) (*Wallet, error)
}
