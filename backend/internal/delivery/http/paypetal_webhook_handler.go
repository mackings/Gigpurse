package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

// PayPetalWebhookHandler receives PayPetal's TrustCore event notifications.
// PayPetal's docs describe no signature-verification mechanism at all — only
// event names appear in prose — so this handler NEVER trusts the inbound
// payload for a money decision. It only uses the payload to look up which
// local agreement to re-check, then re-fetches that agreement from PayPetal
// with our own Bearer token as the one source of truth. A forged POST can at
// worst trigger one wasted authenticated re-fetch; it can't move money on
// its own, since every finalize/settle path re-validates current state
// before acting (see jobUsecase.FinalizeHire, milestoneUsecase.FinalizeFund).
type PayPetalWebhookHandler struct {
	client      paypetal.API
	escrowRepo  domain.EscrowAgreementRepository
	jobUsecase  domain.JobUsecase
	milestoneUC domain.MilestoneUsecase
	notifRepo   domain.NotificationRepository
}

func NewPayPetalWebhookHandler(client paypetal.API, escrowRepo domain.EscrowAgreementRepository, jobUsecase domain.JobUsecase, milestoneUC domain.MilestoneUsecase, notifRepo domain.NotificationRepository) *PayPetalWebhookHandler {
	return &PayPetalWebhookHandler{client: client, escrowRepo: escrowRepo, jobUsecase: jobUsecase, milestoneUC: milestoneUC, notifRepo: notifRepo}
}

// RegisterRoutes is called directly on mux in main.go, deliberately outside
// JWTMiddleware — PayPetal isn't a logged-in GigPurse user.
func (h *PayPetalWebhookHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/webhooks/paypetal", h.Handle)
}

// webhookPayload only extracts the one field this handler actually trusts:
// which reference to go re-check. Every other field PayPetal sends
// (claimed status, amount, etc.) is deliberately ignored.
type webhookPayload struct {
	Reference string `json:"reference"`
	Data      struct {
		Reference string `json:"reference"`
	} `json:"data"`
}

func (h *PayPetalWebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		// Malformed body — nothing to reconcile, but still 200 so PayPetal
		// doesn't retry-storm a request it will never send correctly.
		w.WriteHeader(http.StatusOK)
		return
	}
	reference := payload.Reference
	if reference == "" {
		reference = payload.Data.Reference
	}
	if reference == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.reconcile(r.Context(), reference)
	w.WriteHeader(http.StatusOK)
}

func (h *PayPetalWebhookHandler) reconcile(ctx context.Context, reference string) {
	agreement, err := h.escrowRepo.GetByReference(ctx, reference)
	if err != nil {
		// A reference GigPurse doesn't recognize — log and stop, this is
		// not something retrying will fix.
		log.Printf("paypetal webhook: unknown reference %q", reference)
		return
	}

	// Not yet confirmed funded — finalize the hire/milestone, which itself
	// re-fetches and validates status before acting.
	if agreement.Status != "ONGOING" {
		switch agreement.ScopeType {
		case "job_hire":
			if _, err := h.jobUsecase.FinalizeHire(ctx, reference); err != nil {
				log.Printf("paypetal webhook: FinalizeHire(%s) not yet confirmable: %v", reference, err)
			}
		case "milestone":
			if _, err := h.milestoneUC.FinalizeFund(ctx, reference); err != nil {
				log.Printf("paypetal webhook: FinalizeFund(%s) not yet confirmable: %v", reference, err)
			}
		}
		return
	}

	// Already funded — this event is a payout/refund lifecycle update.
	// Re-fetch as the source of truth and only act on an actual transition
	// (compare old vs new before writing) so a redelivered webhook doesn't
	// double-notify.
	state, err := h.client.GetTrustCoreAgreement(ctx, reference)
	if err != nil {
		log.Printf("paypetal webhook: re-fetch failed for %q: %v", reference, err)
		return
	}

	payoutChanged := state.PayoutStatus != agreement.PayoutStatus
	refundChanged := state.RefundStatus != agreement.RefundStatus
	if !payoutChanged && !refundChanged {
		return
	}

	agreement.PayoutStatus = state.PayoutStatus
	agreement.RefundStatus = state.RefundStatus
	_ = h.escrowRepo.Update(ctx, agreement)

	if payoutChanged && isTerminalStatus(state.PayoutStatus) {
		h.notify(ctx, agreement.CounterpartyUserID, "Payout complete",
			"Your payout has landed in your bank account.")
	}
	if refundChanged && isTerminalStatus(state.RefundStatus) {
		h.notify(ctx, agreement.InitiatorUserID, "Refund complete",
			"Your refund has landed in your bank account.")
	}
}

func isTerminalStatus(status string) bool {
	return status == "COMPLETED" || status == "SUCCESS" || status == "SUCCESSFUL"
}

func (h *PayPetalWebhookHandler) notify(ctx context.Context, userID, title, message string) {
	_ = h.notifRepo.Create(ctx, &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	})
}
