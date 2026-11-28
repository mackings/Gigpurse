package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"gigpurse/internal/domain"
)

// PayPetalWebhookHandler receives PayPetal's TrustVault events, and the
// TrustCore events for automatic payouts.
// It NEVER trusts the inbound payload for a money decision — the payload only
// says which local agreement to re-check, and PayPetal is then asked with our
// own token. A forged POST can at worst trigger one wasted re-fetch.
type PayPetalWebhookHandler struct {
	escrowRepo  domain.EscrowAgreementRepository
	milestoneUC domain.MilestoneUsecase
	payoutUC    domain.PayoutUsecase
}

func NewPayPetalWebhookHandler(escrowRepo domain.EscrowAgreementRepository, milestoneUC domain.MilestoneUsecase, payoutUC domain.PayoutUsecase) *PayPetalWebhookHandler {
	return &PayPetalWebhookHandler{escrowRepo: escrowRepo, milestoneUC: milestoneUC, payoutUC: payoutUC}
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
	// TrustVault release events put PayPetal's own reference in "reference"
	// and ours here.
	IdempotencyKey string `json:"idempotencyKey"`
	Data           struct {
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
	reference := payload.IdempotencyKey
	if reference == "" {
		reference = payload.Reference
	}
	if reference == "" {
		reference = payload.Data.Reference
	}
	if reference == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if domain.IsAutoPayoutReference(reference) {
		// A payout's bank transfer moved on. PayPetal wants an answer within
		// 15 seconds, so check on it (with our own token) after replying.
		go h.payoutUC.ProcessAuto(context.Background())
		w.WriteHeader(http.StatusOK)
		return
	}
	h.reconcile(r.Context(), reference)
	w.WriteHeader(http.StatusOK)
}

func (h *PayPetalWebhookHandler) reconcile(ctx context.Context, reference string) {
	agreement, err := h.escrowRepo.GetByReference(ctx, reference)
	if err != nil || !agreement.IsVault() {
		// A reference GigPurse doesn't recognize — log and stop, this is
		// not something retrying will fix.
		log.Printf("paypetal webhook: unknown reference %q", reference)
		return
	}
	if agreement.Status != "ONGOING" {
		if _, err := h.milestoneUC.FinalizeFund(ctx, reference); err != nil {
			log.Printf("paypetal webhook: FinalizeFund(%s) not yet confirmable: %v", reference, err)
		}
		return
	}
	// Already funded, so this is a release.
	h.milestoneUC.ReconcileVault(ctx)
}
