package usecase

import (
	"context"
	"errors"
	"fmt"

	"gigpurse/internal/domain"
)

type walletUsecase struct {
	walletRepo domain.WalletRepository
}

func NewWalletUsecase(repo domain.WalletRepository) domain.WalletUsecase {
	return &walletUsecase{
		walletRepo: repo,
	}
}

func (u *walletUsecase) GetBalance(ctx context.Context, userID string) (float64, error) {
	w, err := u.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("usecase get balance: %w", err)
	}
	return w.Balance, nil
}

func (u *walletUsecase) Deposit(ctx context.Context, userID string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than zero")
	}

	w, err := u.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("usecase deposit: %w", err)
	}

	return u.walletRepo.UpdateBalance(ctx, w.ID, amount)
}

func (u *walletUsecase) CreateWallet(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user id cannot be empty")
	}

	// Generate a simple unique ID for demonstration
	walletID := fmt.Sprintf("wall_%s", userID)

	newWallet := &domain.Wallet{
		ID:      walletID,
		UserID:  userID,
		Balance: 0.0,
	}

	return u.walletRepo.Create(ctx, newWallet)
}
