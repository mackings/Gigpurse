package usecase

import (
	"context"
	"time"

	"gigpurse/internal/domain"
)

// StartEscrowReconciler is the actual safety net behind payment confirmation:
// PayPetal's webhooks are best-effort, and a user's browser can't be relied on
// to stay open. It re-checks every locally-PENDING payment against PayPetal,
// and settles releases whose outcome wasn't known at the time.
func StartEscrowReconciler(ctx context.Context, escrowRepo domain.EscrowAgreementRepository, milestoneUC domain.MilestoneUsecase, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reconcilePendingEscrow(ctx, escrowRepo, milestoneUC)
				milestoneUC.ReconcileVault(ctx)
			}
		}
	}()
}

func reconcilePendingEscrow(ctx context.Context, escrowRepo domain.EscrowAgreementRepository, milestoneUC domain.MilestoneUsecase) {
	pending, err := escrowRepo.ListPending(ctx)
	if err != nil {
		return
	}
	for _, a := range pending {
		if a.IsVault() {
			// An error just means it isn't paid yet.
			_, _ = milestoneUC.FinalizeFund(ctx, a.ReferenceID)
		}
	}
}
