package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gigpurse/internal/domain"
)

type walletUsecase struct {
	walletRepo       domain.WalletRepository
	userRepo         domain.UserRepository
	escrowRepo       domain.EscrowAgreementRepository
	jobRepo          domain.JobRepository
	milestoneRepo    domain.MilestoneRepository
	jobUsecase       domain.JobUsecase
	milestoneUsecase domain.MilestoneUsecase
}

func NewWalletUsecase(
	repo domain.WalletRepository,
	userRepo domain.UserRepository,
	escrowRepo domain.EscrowAgreementRepository,
	jobRepo domain.JobRepository,
	milestoneRepo domain.MilestoneRepository,
	jobUsecase domain.JobUsecase,
	milestoneUsecase domain.MilestoneUsecase,
) domain.WalletUsecase {
	return &walletUsecase{
		walletRepo:       repo,
		userRepo:         userRepo,
		escrowRepo:       escrowRepo,
		jobRepo:          jobRepo,
		milestoneRepo:    milestoneRepo,
		jobUsecase:       jobUsecase,
		milestoneUsecase: milestoneUsecase,
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
		case "refund":
			// A refund returns money the client already counted as spent
			// via escrow_hold above — net it back out, or "Total spent"
			// stays permanently overstated by every refunded amount.
			summary.TotalSpent -= tx.Amount
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

// ListEscrowHoldings is a client's view of PayPetal's TrustCore: every
// payment they've made that's still actually held (paid in, not yet paid
// out to the other side or refunded back to them), enriched with what it's
// for and who the other party is so the wallet page can list each one with
// its own refund action.
func (u *walletUsecase) ListEscrowHoldings(ctx context.Context, userID string) ([]*domain.EscrowHolding, error) {
	agreements, err := u.escrowRepo.ListByInitiator(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase list escrow holdings: %w", err)
	}

	var holdings []*domain.EscrowHolding
	for _, a := range agreements {
		// Only money genuinely still held counts: PENDING means the
		// checkout was never completed, and any payout/refund status other
		// than NONE means it's already settled away from escrow.
		if a.Status != "ONGOING" || a.PayoutStatus != "NONE" || a.RefundStatus != "NONE" {
			continue
		}

		var title string
		switch a.ScopeType {
		case "job_hire":
			if app, err := u.jobRepo.GetApplicationByID(ctx, a.ScopeID); err == nil {
				if job, err := u.jobRepo.GetByID(ctx, app.JobID); err == nil {
					title = job.Title
				}
			}
		case "milestone":
			if m, err := u.milestoneRepo.GetByID(ctx, a.ScopeID); err == nil {
				title = m.Title
			}
		}
		if title == "" {
			continue // the job/milestone it's scoped to is gone — nothing sensible to show
		}

		var counterpartyName string
		if cp, err := u.userRepo.GetByID(ctx, a.CounterpartyUserID); err == nil {
			counterpartyName = cp.Name
		}

		holdings = append(holdings, &domain.EscrowHolding{
			ReferenceID:      a.ReferenceID,
			ScopeType:        a.ScopeType,
			Title:            title,
			CounterpartyName: counterpartyName,
			Amount:           a.AmountNaira,
			CreatedAt:        a.CreatedAt,
		})
	}

	sort.Slice(holdings, func(i, j int) bool { return holdings[i].CreatedAt.After(holdings[j].CreatedAt) })
	return holdings, nil
}

// RequestRefund routes to whichever usecase actually owns the money-moving
// logic for this holding's scope — wallet itself never talks to PayPetal
// directly, it just picks the right specialist.
func (u *walletUsecase) RequestRefund(ctx context.Context, userID, referenceID string) error {
	agreement, err := u.escrowRepo.GetByReference(ctx, referenceID)
	if err != nil {
		return fmt.Errorf("escrow agreement not found: %w", err)
	}
	switch agreement.ScopeType {
	case "job_hire":
		return u.jobUsecase.RequestHireRefund(ctx, userID, referenceID)
	case "milestone":
		return u.milestoneUsecase.RequestRefund(ctx, userID, referenceID)
	default:
		return errors.New("unknown escrow scope type")
	}
}
