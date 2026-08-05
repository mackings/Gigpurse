package usecase

import (
	"context"
	"fmt"

	"gigpurse/internal/domain"
)

type walletUsecase struct {
	walletRepo domain.WalletRepository
	userRepo   domain.UserRepository
}

func NewWalletUsecase(repo domain.WalletRepository, userRepo domain.UserRepository) domain.WalletUsecase {
	return &walletUsecase{
		walletRepo: repo,
		userRepo:   userRepo,
	}
}

// GetWallet computes total earned/spent from the transaction log (the log
// is the source of truth — nothing mutates a running balance anymore) and
// reports whether a payout account is on file, which is what the Wallet
// page shows instead of a spendable balance.
func (u *walletUsecase) GetWallet(ctx context.Context, userID string) (*domain.WalletSummary, error) {
	txs, err := u.walletRepo.ListTransactions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase get wallet: %w", err)
	}
	summary := &domain.WalletSummary{UserID: userID}
	for _, tx := range txs {
		switch tx.Type {
		case "payment_received":
			summary.TotalEarned += tx.Amount
		case "escrow_hold":
			summary.TotalSpent += tx.Amount
		}
	}
	if user, err := u.userRepo.GetByID(ctx, userID); err == nil {
		summary.HasPayoutAccount = user.PayoutAccount != nil
		summary.PayoutAccount = user.PayoutAccount
	}
	return summary, nil
}

func (u *walletUsecase) ListTransactions(ctx context.Context, userID string) ([]*domain.Transaction, error) {
	return u.walletRepo.ListTransactions(ctx, userID)
}
