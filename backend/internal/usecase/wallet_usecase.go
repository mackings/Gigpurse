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

func (u *walletUsecase) GetWallet(ctx context.Context, userID string) (*domain.Wallet, error) {
	wallet, err := u.walletRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase get wallet: %w", err)
	}
	return wallet, nil
}

func (u *walletUsecase) ListTransactions(ctx context.Context, userID string) ([]*domain.Transaction, error) {
	return u.walletRepo.ListTransactions(ctx, userID)
}

func (u *walletUsecase) Deposit(ctx context.Context, userID string, amount float64) (*domain.Wallet, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	wallet, err := u.walletRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase deposit: %w", err)
	}

	wallet.Balance += amount
	if err := u.walletRepo.Save(ctx, wallet); err != nil {
		return nil, fmt.Errorf("usecase deposit: %w", err)
	}
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: userID, Type: "deposit", Amount: amount, Description: "Wallet top-up",
	})
	return wallet, nil
}

func (u *walletUsecase) Withdraw(ctx context.Context, userID string, amount float64) (*domain.Wallet, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	wallet, err := u.walletRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase withdraw: %w", err)
	}
	if amount > wallet.Balance {
		return nil, errors.New("insufficient balance")
	}

	wallet.Balance -= amount
	if err := u.walletRepo.Save(ctx, wallet); err != nil {
		return nil, fmt.Errorf("usecase withdraw: %w", err)
	}
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: userID, Type: "withdrawal", Amount: amount, Description: "Withdrawal to bank account",
	})
	return wallet, nil
}
