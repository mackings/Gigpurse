package domain

import (
	"context"
	"time"
)

// EscrowAgreement mirrors one PayPetal TrustCore agreement locally, scoped
// to whichever GigPurse concept already has two fixed parties: a whole-job
// hire (ScopeType "job_hire", ScopeID = JobApplication.ID) or a milestone
// (ScopeType "milestone", ScopeID = Milestone.ID). Status/PayoutStatus/
// RefundStatus mirror PayPetal's own vocabulary exactly (PENDING, ONGOING,
// COMPLETED, ...) rather than translating it, since every write to these
// fields comes from re-fetching PayPetal as the source of truth — see
// paypetal_helpers.go and the webhook handler.
type EscrowAgreement struct {
	ID          string `json:"id" bson:"_id"`
	ReferenceID string `json:"reference_id" bson:"reference_id"` // "job-hire:{applicationID}" or "milestone:{milestoneID}"
	ScopeType   string `json:"scope_type" bson:"scope_type"`     // "job_hire" | "milestone"
	ScopeID     string `json:"scope_id" bson:"scope_id"`
	ContractID  string `json:"contract_id,omitempty" bson:"contract_id,omitempty"`

	// PayPetal-side identities — never serialized to the frontend.
	InitiatorCustomerID    string `json:"-" bson:"initiator_customer_id"`
	CounterpartyCustomerID string `json:"-" bson:"counterparty_customer_id"`

	InitiatorUserID    string  `json:"initiator_user_id" bson:"initiator_user_id"`
	CounterpartyUserID string  `json:"counterparty_user_id" bson:"counterparty_user_id"`
	AmountNaira        float64 `json:"amount" bson:"amount_naira"`

	TrustCoreTransactionID string `json:"-" bson:"trustcore_transaction_id"`

	Status       string `json:"status" bson:"status"`               // PENDING, ONGOING, ...
	PayoutStatus string `json:"payout_status" bson:"payout_status"` // NONE, PENDING, COMPLETED, ...
	RefundStatus string `json:"refund_status" bson:"refund_status"` // NONE, PENDING, COMPLETED, ...

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type EscrowAgreementRepository interface {
	Create(ctx context.Context, a *EscrowAgreement) error
	GetByReference(ctx context.Context, referenceID string) (*EscrowAgreement, error)
	Update(ctx context.Context, a *EscrowAgreement) error
	// ListStalePending finds agreements still awaiting payment past a
	// cutoff, for the abandoned-checkout sweep to revert.
	ListStalePending(ctx context.Context, olderThan time.Time) ([]*EscrowAgreement, error)
	// ListByInitiator finds every agreement this user paid into — the
	// client's wallet page sums these into "currently in escrow".
	ListByInitiator(ctx context.Context, userID string) ([]*EscrowAgreement, error)
}
