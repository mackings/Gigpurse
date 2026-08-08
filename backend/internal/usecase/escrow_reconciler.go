package usecase

import (
	"context"
	"time"

	"gigpurse/internal/domain"
)

// StartEscrowReconciler is the actual safety net behind payment
// confirmation. PayPetal's docs describe webhook events only in prose —
// there's no documented endpoint to register a webhook URL, no way to
// verify one is even configured, and CreateTrustCoreAgreement never sends
// one — so whether PayPetal ever calls /webhooks/paypetal at all is
// unverified. Confirmation can't be allowed to depend solely on that, or on
// a user's browser staying on /contracts/pending long enough for its own
// short poll to succeed (checkout redirects, popup closes, connection
// drops — none of that should mean a real payment silently never lands).
//
// This periodically re-checks every locally-PENDING agreement directly
// against PayPetal and finalizes whichever have actually gone ONGOING —
// same effect as the webhook firing, just on a timer instead of a push.
// jobUsecase's own abandoned-hire sweep (and the equivalent for milestones)
// still separately reverts ones that are genuinely never completed.
func StartEscrowReconciler(ctx context.Context, escrowRepo domain.EscrowAgreementRepository, jobUC domain.JobUsecase, milestoneUC domain.MilestoneUsecase, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reconcilePendingEscrow(ctx, escrowRepo, jobUC, milestoneUC)
			}
		}
	}()
}

func reconcilePendingEscrow(ctx context.Context, escrowRepo domain.EscrowAgreementRepository, jobUC domain.JobUsecase, milestoneUC domain.MilestoneUsecase) {
	pending, err := escrowRepo.ListPending(ctx)
	if err != nil {
		return
	}
	for _, a := range pending {
		switch a.ScopeType {
		case "job_hire":
			// Errors here just mean PayPetal still reports it PENDING (not
			// yet paid, or genuinely abandoned) — nothing to do until either
			// the client pays or the abandoned-hire sweep reverts it.
			_, _ = jobUC.FinalizeHire(ctx, a.ReferenceID)
		case "milestone":
			_, _ = milestoneUC.FinalizeFund(ctx, a.ReferenceID)
		}
	}
}
