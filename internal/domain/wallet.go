package domain

import "context"

type Wallet struct {
	ID      string  `json:"id"`
	UserID  string  `json:"user_id"`
	Balance float64 `json:"balance"`
}

type WalletRepository interface {
	GetByUserID(ctx context.Context, userID string) (*Wallet, error)
	UpdateBalance(ctx context.Context, walletID string, amount float64) error
	Create(ctx context.Context, wallet *Wallet) error
}

type WalletUsecase interface {
	GetBalance(ctx context.Context, userID string) (float64, error)
	Deposit(ctx context.Context, userID string, amount float64) error
	CreateWallet(ctx context.Context, userID string) error
}
