package domain

import (
	"context"
	"time"
)

// Milestone status lifecycle:
//
//	proposed  -- the client proposes a milestone for a contract; the talent
//	             can accept, reject, or counter it (a counter keeps the
//	             status at "proposed" but flips who's offering what — from
//	             then on it's whoever didn't make the last offer who can
//	             respond, so a countered proposal bounces back to the client)
//	accepted  -- the other party accepted the current terms, making it fundable
//	rejected  -- the other party rejects it (terminal)
//	funded    -- the client funds escrow for an accepted milestone
//	released  -- the client releases escrow, crediting the musician's wallet (terminal)
//	refunded  -- a dispute resolved in the client's favor, so held escrow
//	             went back to their wallet balance instead of the musician's (terminal)
//	cancelled -- either party ended the contract or pulled this milestone
//	             while it was still just "accepted" — no money had moved yet,
//	             so this is a clean cancel, no dispute needed (terminal)
//	disputed  -- a dispute was opened over a `funded` milestone (directly, by
//	             ending the contract, or by withdrawing the milestone) —
//	             locked from Release/RequestRefund/auto-release until a
//	             moderator resolves that dispute
//	settling  -- a moderator has ruled, and the client still
//	             has to enter their emailed code before the money can move
type Milestone struct {
	ID         string                      `json:"id" bson:"_id"`
	ContractID string                      `json:"contract_id" bson:"contract_id"`
	Title      string                      `json:"title" bson:"title"`
	Amount     float64                     `json:"amount" bson:"amount"`
	DueDate    *time.Time                  `json:"due_date,omitempty" bson:"due_date,omitempty"`
	Status     string                      `json:"status" bson:"status"`
	ProposedBy string                      `json:"proposed_by" bson:"proposed_by"`
	History    []MilestoneNegotiationEntry `json:"history,omitempty" bson:"history,omitempty"`
	Order      int                         `json:"order" bson:"order"`
	CreatedAt  time.Time                   `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time                   `json:"updated_at" bson:"updated_at"`

	// LastReminderAt tracks the last time the awaiting-response party was
	// re-notified about this still-`proposed` milestone — nil until the
	// first reminder fires. See MilestoneUsecase reminder scanner.
	LastReminderAt *time.Time `json:"last_reminder_at,omitempty" bson:"last_reminder_at,omitempty"`

	// EscrowReference points at the EscrowAgreement backing this milestone's
	// PayPetal TrustVault payment — set by Fund, confirmed by FinalizeFund.
	EscrowReference string `json:"-" bson:"escrow_reference,omitempty"`
	// EscrowProvider mirrors the agreement's Provider so the frontend knows
	// whether releasing needs a code.
	EscrowProvider string `json:"escrow_provider,omitempty" bson:"escrow_provider,omitempty"`
	// SettlementTalent is what a moderator awarded the talent on a "settling"
	// milestone; the client gets the rest back.
	SettlementTalent *float64 `json:"settlement_talent,omitempty" bson:"settlement_talent,omitempty"`
	// ReleaseNudgedAt is the last time a client was reminded about an
	// overdue TrustVault release request.
	ReleaseNudgedAt *time.Time `json:"-" bson:"release_nudged_at,omitempty"`

	// DisputeID does double duty: on an original milestone it's set the
	// moment a dispute over it opens (locks it at status "disputed"); on a
	// synthetic "Dispute settlement" milestone created by ResolveDispute
	// it links back to the dispute that ordered the payment, so
	// FinalizeFund knows to auto-release it (no separate Release click —
	// a moderator already adjudicated it) and so a client can be blocked
	// from new activity while one sits unfunded. See DisputeUsecase.
	DisputeID string `json:"dispute_id,omitempty" bson:"dispute_id,omitempty"`

	// ReleaseRequestedAt is set when the talent asks for a `funded`
	// milestone to be released — the auto-release scanner pays it out
	// after 48h if the client hasn't acted (released, refunded, or
	// disputed) by then. See MilestoneUsecase.RequestRelease.
	ReleaseRequestedAt *time.Time `json:"release_requested_at,omitempty" bson:"release_requested_at,omitempty"`
}

// MilestoneHoldsMoney reports whether a milestone in this status still has
// the client's money in escrow.
func MilestoneHoldsMoney(status string) bool {
	return status == "funded" || status == "disputed" || status == "settling"
}

// AwaitingClientSettlement reports whether a moderator has ruled on this and
// the client still owes the next step: confirming the ruling with their code
// (or paying a settlement ordered before TrustVault).
func (m *Milestone) AwaitingClientSettlement() bool {
	return m.Status == "settling" || (m.Status == "accepted" && m.DisputeID != "")
}

// MilestoneNegotiationEntry records one offer in a milestone's back-and-forth
// — every propose/counter appends one, so both parties can see who offered
// what terms, mirroring how direct-hire booking negotiation works.
type MilestoneNegotiationEntry struct {
	ProposedBy string     `json:"proposed_by" bson:"proposed_by"`
	Title      string     `json:"title,omitempty" bson:"title,omitempty"`
	Amount     float64    `json:"amount" bson:"amount"`
	DueDate    *time.Time `json:"due_date,omitempty" bson:"due_date,omitempty"`
	CreatedAt  time.Time  `json:"created_at" bson:"created_at"`
}

type MilestoneInput struct {
	Title   string     `json:"title"`
	Amount  float64    `json:"amount"`
	DueDate *time.Time `json:"due_date,omitempty"`
}

type MilestoneRepository interface {
	Create(ctx context.Context, m *Milestone) error
	GetByID(ctx context.Context, id string) (*Milestone, error)
	ListByContract(ctx context.Context, contractID string) ([]*Milestone, error)
	Update(ctx context.Context, m *Milestone) error
	Delete(ctx context.Context, id string) error
	// ListByStatus lists across every contract — used by the reminder
	// scanner to find every still-`proposed` milestone system-wide.
	ListByStatus(ctx context.Context, status string) ([]*Milestone, error)
	// CompareAndSwapStatus atomically flips status from expectedStatus to
	// newStatus only if it's still expectedStatus at the moment of the
	// write, and reports whether this call was the one that made the
	// change. This is the concurrency guard around every irreversible
	// status transition that a real-world race can trigger twice — most
	// notably PayPetal's webhook and GigPurse's own reconciler both calling
	// FinalizeFund for the same payment within moments of each other
	// (confirmed live: both slipped past a plain status == "accepted"
	// check before either had written, double-recorded the funding
	// transaction, and one silently stomped the other's "released" back to
	// "funded"). The loser should treat `false, nil` as "someone else
	// already handled this," not as an error — never retry the same work.
	CompareAndSwapStatus(ctx context.Context, id, expectedStatus, newStatus string) (bool, error)
}

type MilestoneUsecase interface {
	Propose(ctx context.Context, contractID, proposerID string, items []MilestoneInput) ([]*Milestone, error)
	Accept(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Reject(ctx context.Context, contractID, milestoneID, userID string) (*Milestone, error)
	Counter(ctx context.Context, contractID, milestoneID, userID string, terms MilestoneInput) (*Milestone, error)
	// Withdraw lets the proposer retract their own still-pending proposal
	// (e.g. they mistyped an amount or date) so they can send a corrected
	// one — only while it's awaiting a response, before it clutters the
	// history with a rejected/superseded entry.
	Withdraw(ctx context.Context, contractID, milestoneID, userID string) error
	// Fund starts a PayPetal TrustVault payment for an accepted milestone and
	// returns a hosted checkout URL — it no longer moves money itself or
	// flips the milestone to "funded" synchronously; FinalizeFund does that
	// once the payment is confirmed (by the webhook or a frontend poll).
	Fund(ctx context.Context, contractID, milestoneID, userID string) (paymentURL, reference string, err error)
	FinalizeFund(ctx context.Context, reference string) (*Milestone, error)
	// Release pays out a milestone with the code PayPetal emailed the client.
	Release(ctx context.Context, contractID, milestoneID, userID, code string) (*Milestone, error)
	// List returns a contract's milestones for a client/musician participant,
	// or for an admin/moderator (requesterRole) reviewing a dispute — pass
	// "" for requesterRole from any ordinary participant-only call site.
	List(ctx context.Context, contractID, requesterID, requesterRole string) ([]*Milestone, error)

	// TrustVault payments only move with a 6-digit code PayPetal emails the
	// client. SendMilestoneReleaseCode/SendReleaseCode trigger that email;
	// Release and RequestRefund use the code.
	SendMilestoneReleaseCode(ctx context.Context, contractID, milestoneID, userID string) error
	SendReleaseCode(ctx context.Context, userID, referenceID string) error
	// BeginVaultSettlement records a moderator's ruling on a disputed
	// TrustVault milestone — never exposed on any HTTP route.
	BeginVaultSettlement(ctx context.Context, milestoneID string, talentAward float64) error
	// ReconcileVault finishes or undoes releases PayPetal hadn't confirmed.
	ReconcileVault(ctx context.Context)

	// RequestRefund returns a funded payment to the client, with their code.
	RequestRefund(ctx context.Context, clientID, referenceID, code string) error

	// CancelAccepted cancels a still-`accepted` (not yet funded) milestone —
	// no money involved, so no dispute needed. Ungated like
	// RefundHeldForContract: the caller (EndContract/CancelMilestone) has
	// already authorized the acting user.
	CancelAccepted(ctx context.Context, milestoneID string) error

	// MarkDisputed flips a `funded` milestone to `disputed`, locking it out
	// of Release/RequestRefund while the dispute (disputeID) just opened on
	// it is pending. Ungated for the same reason as CancelAccepted.
	MarkDisputed(ctx context.Context, milestoneID, disputeID string) error

	// RequestRelease lets the talent ask for a `funded` milestone to be paid
	// out — the client is reminded if they leave it unanswered.
	RequestRelease(ctx context.Context, contractID, milestoneID, talentID string) error

	// StartReminderScanner runs in the background for the lifetime of ctx,
	// periodically re-notifying whoever hasn't responded to a still-`proposed`
	// milestone — first after nudgeAfter, then at most every repeatAfter.
	// Called once at startup from main.go.
	StartReminderScanner(ctx context.Context, checkInterval, nudgeAfter, repeatAfter time.Duration)

	// StartReleaseNudgeScanner reminds clients about unanswered release
	// requests. Called once at startup from main.go.
	StartReleaseNudgeScanner(ctx context.Context, checkInterval time.Duration)
}
