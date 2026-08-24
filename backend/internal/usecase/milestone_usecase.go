package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

// Broadcaster is the subset of the websocket Hub that milestoneUsecase needs
// to push a milestone system message live — defined here (consumer side)
// rather than imported from delivery/http, which would create an import
// cycle. main.go wires the concrete *delivery.Hub in, which satisfies this
// interface structurally without either package needing to import the
// other (same pattern as PresenceChecker in user_usecase.go).
type Broadcaster interface {
	Send(userID string, msgType string, data interface{}) bool
}

type milestoneUsecase struct {
	paypetalDeps
	milestoneRepo   domain.MilestoneRepository
	contractRepo    domain.ContractRepository
	walletRepo      domain.WalletRepository
	notifRepo       domain.NotificationRepository
	chatRepo        domain.ChatRepository
	broadcaster     Broadcaster
	escrowRepo      domain.EscrowAgreementRepository
	frontendBaseURL string
}

func NewMilestoneUsecase(
	milestoneRepo domain.MilestoneRepository,
	contractRepo domain.ContractRepository,
	walletRepo domain.WalletRepository,
	notifRepo domain.NotificationRepository,
	chatRepo domain.ChatRepository,
	broadcaster Broadcaster,
	paypetalClient paypetal.API,
	userRepo domain.UserRepository,
	escrowRepo domain.EscrowAgreementRepository,
	frontendBaseURL string,
) domain.MilestoneUsecase {
	return &milestoneUsecase{
		paypetalDeps:    paypetalDeps{client: paypetalClient, userRepo: userRepo},
		milestoneRepo:   milestoneRepo,
		contractRepo:    contractRepo,
		walletRepo:      walletRepo,
		notifRepo:       notifRepo,
		chatRepo:        chatRepo,
		broadcaster:     broadcaster,
		escrowRepo:      escrowRepo,
		frontendBaseURL: frontendBaseURL,
	}
}

// postMilestoneChatMessage drops an automatic message into the two parties'
// normal 1:1 chat thread (same mechanism dispute rooms use for "a moderator
// joined") so a proposed/countered milestone is visible in Messages, not
// just as a notification, and pushes it live over the socket to both sides.
// ContractID/MilestoneID are set on the message so the frontend can render
// an actual actionable milestone card inline instead of plain text — a
// message that just says "review it" with no way to act on it right there
// is not meaningfully different from the notification alone.
func (u *milestoneUsecase) postMilestoneChatMessage(ctx context.Context, senderID, recvID, contractID, milestoneID, content string) {
	msg := &domain.ChatMessage{
		SenderID:    senderID,
		RecvID:      recvID,
		IsSystem:    true,
		Content:     content,
		Timestamp:   time.Now(),
		ContractID:  contractID,
		MilestoneID: milestoneID,
	}
	if err := u.chatRepo.SaveMessage(ctx, msg); err != nil {
		return
	}
	u.broadcaster.Send(senderID, "chat_message", msg)
	u.broadcaster.Send(recvID, "chat_message", msg)
}

// participant checks the requester is one of the two parties on the
// contract and returns the counterparty's user ID (who every milestone
// notification for this contract is addressed to/from).
func (u *milestoneUsecase) participant(contract *domain.Contract, userID string) (counterpart string, ok bool) {
	switch userID {
	case contract.ClientID:
		return contract.MusicianID, true
	case contract.MusicianID:
		return contract.ClientID, true
	default:
		return "", false
	}
}

func (u *milestoneUsecase) notify(ctx context.Context, userID, title, message, contractID string) {
	_ = u.notifRepo.Create(ctx, &domain.Notification{
		UserID:     userID,
		Title:      title,
		Message:    message,
		ContractID: contractID,
		CreatedAt:  time.Now(),
	})
	if user, err := u.userRepo.GetByID(ctx, userID); err == nil && user.Email != "" {
		if err := sendEmailFn(user.Email, title, message); err != nil {
			log.Printf("notify: email to %s failed: %v", user.Email, err)
		}
	}
}

func (u *milestoneUsecase) Propose(ctx context.Context, contractID, proposerID string, items []domain.MilestoneInput) ([]*domain.Milestone, error) {
	if len(items) == 0 {
		return nil, errors.New("at least one milestone is required")
	}
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	counterpart, ok := u.participant(contract, proposerID)
	if !ok {
		return nil, errors.New("unauthorized: not a participant on this contract")
	}
	if proposerID != contract.ClientID {
		return nil, errors.New("unauthorized: only the client can propose a new milestone — the talent can counter, accept, or reject one")
	}
	if contract.Status == "disputed" {
		return nil, errors.New("this contract has a dispute pending resolution — new milestones can't be proposed until it's resolved")
	}

	// Proposing new work reopens a contract the client had marked
	// completed (or that got cancelled) — treated as "there's more to do
	// here" rather than requiring a whole separate contract for it.
	reopened := contract.Status != "active"
	if reopened {
		contract.Status = "active"
		contract.UpdatedAt = time.Now()
		if err := u.contractRepo.Update(ctx, contract); err != nil {
			return nil, fmt.Errorf("failed to reopen contract: %w", err)
		}
	}

	existing, err := u.milestoneRepo.ListByContract(ctx, contractID)
	if err != nil {
		return nil, err
	}

	var created []*domain.Milestone
	for i, item := range items {
		if item.Title == "" || item.Amount <= 0 {
			return nil, errors.New("each milestone needs a title and an amount greater than zero")
		}
		now := time.Now()
		m := &domain.Milestone{
			ContractID: contractID,
			Title:      item.Title,
			Amount:     item.Amount,
			DueDate:    item.DueDate,
			Status:     "proposed",
			ProposedBy: proposerID,
			History: []domain.MilestoneNegotiationEntry{
				{ProposedBy: proposerID, Title: item.Title, Amount: item.Amount, DueDate: item.DueDate, CreatedAt: now},
			},
			Order: len(existing) + i,
		}
		if err := u.milestoneRepo.Create(ctx, m); err != nil {
			return nil, err
		}
		created = append(created, m)
	}

	if reopened {
		proposerName := "The client"
		if proposerUser, err := u.userRepo.GetByID(ctx, proposerID); err == nil && proposerUser.Name != "" {
			proposerName = proposerUser.Name
		}
		u.notify(ctx, counterpart, "Contract reopened",
			fmt.Sprintf("%s proposed a new milestone on '%s' — the contract is active again.", proposerName, contract.Title), contractID)
	}
	u.notify(ctx, counterpart, "New milestone proposed",
		fmt.Sprintf("A new milestone was proposed: review it in your contract chat."), contractID)

	// One chat message per milestone (not a combined summary) so each one
	// carries its own MilestoneID and renders as its own actionable card.
	for _, m := range created {
		u.postMilestoneChatMessage(ctx, proposerID, counterpart, contractID, m.ID,
			fmt.Sprintf("Milestone proposed: '%s' (%s).", m.Title, formatNaira(m.Amount)))
	}

	return created, nil
}

func (u *milestoneUsecase) loadForTransition(ctx context.Context, contractID, milestoneID, userID string) (*domain.Contract, *domain.Milestone, string, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("contract not found: %w", err)
	}
	counterpart, ok := u.participant(contract, userID)
	if !ok {
		return nil, nil, "", errors.New("unauthorized: not a participant on this contract")
	}
	milestone, err := u.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil || milestone.ContractID != contractID {
		return nil, nil, "", errors.New("milestone not found")
	}
	return contract, milestone, counterpart, nil
}

func (u *milestoneUsecase) Accept(ctx context.Context, contractID, milestoneID, userID string) (*domain.Milestone, error) {
	_, milestone, counterpart, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return nil, err
	}
	if milestone.ProposedBy == userID {
		return nil, errors.New("you proposed this milestone; the other party must accept it")
	}
	if milestone.Status != "proposed" {
		return nil, errors.New("milestone is not awaiting a response")
	}

	milestone.Status = "accepted"
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return nil, err
	}
	// counterpart here is the other party relative to the accepter, i.e. the
	// original proposer — notify them their milestone was accepted.
	u.notify(ctx, counterpart, "Milestone accepted",
		fmt.Sprintf("Your milestone '%s' (%s) was accepted.", milestone.Title, formatNaira(milestone.Amount)), contractID)

	return milestone, nil
}

func (u *milestoneUsecase) Reject(ctx context.Context, contractID, milestoneID, userID string) (*domain.Milestone, error) {
	_, milestone, counterpart, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return nil, err
	}
	if milestone.ProposedBy == userID {
		return nil, errors.New("you proposed this milestone; the other party must respond to it")
	}
	if milestone.Status != "proposed" {
		return nil, errors.New("milestone is not awaiting a response")
	}

	milestone.Status = "rejected"
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return nil, err
	}
	u.notify(ctx, counterpart, "Milestone rejected",
		fmt.Sprintf("Your milestone '%s' (%s) was rejected.", milestone.Title, formatNaira(milestone.Amount)), contractID)

	return milestone, nil
}

func (u *milestoneUsecase) Withdraw(ctx context.Context, contractID, milestoneID, userID string) error {
	_, milestone, counterpart, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return err
	}
	if milestone.ProposedBy != userID {
		return errors.New("only the party who proposed this milestone can withdraw it")
	}
	if milestone.Status != "proposed" {
		return errors.New("only a still-pending milestone can be withdrawn")
	}
	if err := u.milestoneRepo.Delete(ctx, milestoneID); err != nil {
		return err
	}
	u.notify(ctx, counterpart, "Milestone withdrawn",
		fmt.Sprintf("The milestone '%s' was withdrawn by the other party.", milestone.Title), contractID)
	return nil
}

func (u *milestoneUsecase) Counter(ctx context.Context, contractID, milestoneID, userID string, terms domain.MilestoneInput) (*domain.Milestone, error) {
	if terms.Amount <= 0 {
		return nil, errors.New("counter-offer needs a positive amount")
	}
	_, milestone, counterpart, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return nil, err
	}
	if milestone.ProposedBy == userID {
		return nil, errors.New("you made the current offer; the other party must respond to it")
	}
	if milestone.Status != "proposed" {
		return nil, errors.New("milestone is not awaiting a response")
	}

	now := time.Now()
	if terms.Title != "" {
		milestone.Title = terms.Title
	}
	milestone.Amount = terms.Amount
	milestone.DueDate = terms.DueDate
	milestone.ProposedBy = userID
	milestone.History = append(milestone.History, domain.MilestoneNegotiationEntry{
		ProposedBy: userID,
		Title:      milestone.Title,
		Amount:     milestone.Amount,
		DueDate:    milestone.DueDate,
		CreatedAt:  now,
	})
	milestone.UpdatedAt = now
	// A counter-offer restarts the response clock — the reminder scanner
	// should wait a fresh 5 minutes from this new offer, not still be
	// counting from whenever the original proposal went out.
	milestone.LastReminderAt = nil
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return nil, err
	}
	u.notify(ctx, counterpart, "Milestone Terms Updated",
		fmt.Sprintf("New offer for '%s': %s", milestone.Title, formatNaira(milestone.Amount)), contractID)
	u.postMilestoneChatMessage(ctx, userID, counterpart, contractID, milestone.ID,
		fmt.Sprintf("New milestone offer: '%s' — %s.", milestone.Title, formatNaira(milestone.Amount)))

	return milestone, nil
}

// Fund starts a TrustCore payment for an accepted milestone and returns a
// hosted checkout URL — see FinalizeFund for what used to happen here
// synchronously.
func (u *milestoneUsecase) Fund(ctx context.Context, contractID, milestoneID, userID string) (string, string, error) {
	contract, milestone, _, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return "", "", err
	}
	if userID != contract.ClientID {
		return "", "", errors.New("unauthorized: only the client can fund a milestone")
	}
	if milestone.Status != "accepted" {
		return "", "", errors.New("milestone must be accepted by both parties before it can be funded")
	}

	client, err := u.userRepo.GetByID(ctx, contract.ClientID)
	if err != nil {
		return "", "", fmt.Errorf("client not found: %w", err)
	}
	musician, err := u.userRepo.GetByID(ctx, contract.MusicianID)
	if err != nil {
		return "", "", fmt.Errorf("musician not found: %w", err)
	}
	if err := u.requirePayoutAccount(musician); err != nil {
		u.notify(ctx, musician.ID, "Add your bank account to get paid",
			fmt.Sprintf("A milestone payment for '%s' is waiting on you to add a payout account.", milestone.Title), contractID)
		return "", "", err
	}

	clientCustomerID, err := u.ensureCustomer(ctx, client)
	if err != nil {
		return "", "", fmt.Errorf("failed to register client with payment provider: %w", err)
	}
	musicianCustomerID, err := u.ensureCustomer(ctx, musician)
	if err != nil {
		return "", "", fmt.Errorf("failed to register musician with payment provider: %w", err)
	}

	// milestone.Amount is the agreed price — what the talent expects for
	// this piece of work. GigPurse's cut can't be deducted from the
	// release (PayPetal always releases the full escrowed amount), so only
	// the talent's net take-home is actually escrowed; GigPurse's combined
	// commission + service fee rides along as a separate merchantCharge
	// billed to the client on top of it. See pricing.go for the split.
	//
	// A "Dispute settlement" milestone (DisputeID set) is exempt — GigPurse
	// already took its cut on the original funding this is making up for,
	// so charging commission again on the makeup payment would double-dip
	// on a situation that already went wrong. The talent gets the full
	// moderator-awarded amount.
	talentAmount := TalentTakeHome(milestone.Amount)
	platformFee := PlatformFee(milestone.Amount)
	if milestone.DisputeID != "" {
		talentAmount = milestone.Amount
		platformFee = 0
	}

	var merchantChargeKobo string
	if platformFee > 0 {
		merchantChargeKobo = paypetal.NairaToKobo(platformFee)
	}

	// PayPetal's hosted checkout prepends its own "MCH-" prefix before
	// generating a bank-transfer virtual account, and rejects anything past
	// 35 chars at that point (confirmed live: "milestone:" + a 24-char
	// Mongo ID is 34 chars, +4 for "MCH-" = 38, over the limit — every
	// bank-transfer milestone payment 400'd on PayPetal's /pay/create-account
	// step). Keep this prefix short enough to leave headroom.
	reference := "ms:" + milestoneID
	result, err := u.client.CreateTrustCoreAgreement(ctx, paypetal.CreateAgreementInput{
		ReferenceID:            reference,
		InitiatorCustomerID:    clientCustomerID,
		CounterpartyCustomerID: musicianCustomerID,
		Currency:               "NGN",
		AmountKobo:             paypetal.NairaToKobo(talentAmount),
		MerchantChargeKobo:     merchantChargeKobo,
		Description:            fmt.Sprintf("GigPurse milestone: %s", milestone.Title),
		// PayPetal appends "?status=...&txnId=..." to this URL itself, blindly
		// (no check for an existing "?") — so reference must ride in the path,
		// never a query string, or we get a broken double-"?" redirect.
		RedirectURL: u.frontendBaseURL + "/contracts/pending/" + url.PathEscape(reference),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to start payment: %w", err)
	}

	agreement := &domain.EscrowAgreement{
		ReferenceID:            reference,
		ScopeType:              "milestone",
		ScopeID:                milestoneID,
		ContractID:             contractID,
		InitiatorCustomerID:    clientCustomerID,
		CounterpartyCustomerID: musicianCustomerID,
		InitiatorUserID:        contract.ClientID,
		CounterpartyUserID:     contract.MusicianID,
		AmountNaira:            talentAmount,
		PlatformFeeNaira:       platformFee,
		TrustCoreTransactionID: result.TransactionID,
		Status:                 result.Status,
		PayoutStatus:           "NONE",
		RefundStatus:           "NONE",
	}
	if err := u.escrowRepo.Create(ctx, agreement); err != nil {
		return "", "", fmt.Errorf("failed to record escrow agreement: %w", err)
	}

	milestone.EscrowReference = reference
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return "", "", err
	}

	return result.PaymentURL, reference, nil
}

// FinalizeFund confirms a milestone payment and flips it to "funded" — the
// counterpart to job hiring's FinalizeHire. Called from the webhook handler
// and from a frontend poll; idempotent against both firing.
func (u *milestoneUsecase) FinalizeFund(ctx context.Context, reference string) (*domain.Milestone, error) {
	agreement, err := u.escrowRepo.GetByReference(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("escrow agreement not found: %w", err)
	}
	milestone, err := u.milestoneRepo.GetByID(ctx, agreement.ScopeID)
	if err != nil {
		return nil, fmt.Errorf("milestone not found: %w", err)
	}
	if milestone.Status == "funded" || milestone.Status == "released" {
		return milestone, nil // already finalized (released covers a settlement milestone's auto-release below)
	}

	state, err := u.client.GetTrustCoreAgreement(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm payment: %w", err)
	}
	if state.Status != "ONGOING" {
		return nil, fmt.Errorf("payment not yet confirmed (status: %s)", state.Status)
	}

	// Claim this milestone before recording anything — PayPetal's webhook
	// and GigPurse's own reconciler sweep can both reach this point for the
	// same payment within moments of each other (confirmed live: both read
	// "accepted" before either had written "funded", then both proceeded
	// to double-record the funding transaction, and one silently stomped
	// the other's later "released" back down to "funded"). Only the
	// winner of this atomic swap records anything below.
	won, err := u.milestoneRepo.CompareAndSwapStatus(ctx, milestone.ID, "accepted", "funded")
	if err != nil {
		return nil, err
	}
	if !won {
		return u.milestoneRepo.GetByID(ctx, milestone.ID)
	}
	milestone.Status = "funded"

	agreement.Status = state.Status
	agreement.PayoutStatus = state.PayoutStatus
	agreement.RefundStatus = state.RefundStatus
	_ = u.escrowRepo.Update(ctx, agreement)

	// The client's actual charge is amount + platform fee — milestone.Amount
	// is just the agreed price, not what left their payment method.
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: agreement.InitiatorUserID, Type: "escrow_hold", Amount: agreement.AmountNaira + agreement.PlatformFeeNaira,
		Description: fmt.Sprintf("Escrow funded: %s", milestone.Title), Reference: reference,
	})
	u.notify(ctx, agreement.CounterpartyUserID, "Escrow funded",
		fmt.Sprintf("Escrow funded for milestone '%s' (%s).", milestone.Title, formatNaira(milestone.Amount)), milestone.ContractID)

	// A "Dispute settlement" milestone has nothing left to adjudicate — a
	// moderator already ruled on whether the work was done, so it pays out
	// the instant the client funds it rather than waiting on a separate
	// Release click that could just never come.
	if milestone.DisputeID != "" {
		if released, err := u.Release(ctx, milestone.ContractID, milestone.ID, agreement.InitiatorUserID); err == nil {
			return released, nil
		}
	}

	return milestone, nil
}

func (u *milestoneUsecase) Release(ctx context.Context, contractID, milestoneID, userID string) (*domain.Milestone, error) {
	contract, milestone, _, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return nil, err
	}
	if userID != contract.ClientID {
		return nil, errors.New("unauthorized: only the client can release a milestone")
	}
	if milestone.Status != "funded" {
		return nil, errors.New("milestone is not funded yet")
	}
	return u.releaseMilestone(ctx, contract, milestone)
}

// ReleaseDisputed is Release's ungated counterpart for a moderator's
// full-release ruling — the milestone is "disputed" by then (see
// MarkDisputed), not "funded", so Release's own client-facing gate would
// reject it. Deliberately not exposed on any HTTP route: allowing a client
// to call the "funded" version on a disputed milestone directly would be
// harmless (they're just paying early), but allowing them to reach
// RequestRefund's equivalent on a disputed milestone would let them
// self-refund and bypass the moderator entirely — so this stays internal,
// same as RefundMilestone/CancelAccepted/MarkDisputed.
func (u *milestoneUsecase) ReleaseDisputed(ctx context.Context, contractID, milestoneID string) (*domain.Milestone, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	milestone, err := u.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil || milestone.ContractID != contractID {
		return nil, errors.New("milestone not found")
	}
	if milestone.Status != "disputed" {
		return nil, fmt.Errorf("milestone is %s, not disputed", milestone.Status)
	}
	return u.releaseMilestone(ctx, contract, milestone)
}

// releaseMilestone is Release/ReleaseDisputed's shared core once status has
// already been validated by the caller.
func (u *milestoneUsecase) releaseMilestone(ctx context.Context, contract *domain.Contract, milestone *domain.Milestone) (*domain.Milestone, error) {
	if milestone.EscrowReference == "" {
		return nil, errors.New("milestone has no escrow agreement on file")
	}

	// Claim the release before calling PayPetal, not after — a client's
	// manual Release click can race FinalizeFund's own auto-release for a
	// dispute settlement milestone, and without this, both calls reach
	// PayPetal. Confirmed live: PayPetal's second call came back
	// "payout_in_progress," but by then our own status write had already
	// been lost to the other call's overwrite, leaving the milestone stuck
	// at "funded" with no release transaction ever recorded for the talent.
	//
	// The starting status is whatever the caller already validated
	// (Release requires "funded"; ReleaseDisputed requires "disputed") —
	// hardcoding "funded" here silently no-ops ReleaseDisputed forever,
	// since a disputed milestone's status is never "funded" by this point.
	won, err := u.milestoneRepo.CompareAndSwapStatus(ctx, milestone.ID, milestone.Status, "released")
	if err != nil {
		return nil, err
	}
	if !won {
		return u.milestoneRepo.GetByID(ctx, milestone.ID)
	}

	if err := u.client.CompleteTrustCoreAgreement(ctx, milestone.EscrowReference); err != nil {
		if !isPayoutInProgress(err) {
			// Real failure — release the claim so this can be retried
			// (by the client clicking again, or the next reconciler pass),
			// back to whatever status it actually started from.
			_, _ = u.milestoneRepo.CompareAndSwapStatus(ctx, milestone.ID, "released", milestone.Status)
			return nil, fmt.Errorf("failed to release payment: %w", err)
		}
		// An earlier attempt already put this payout in motion on
		// PayPetal's side — that's a real success, not an error, so fall
		// through and record it as one.
	}
	// agreement.AmountNaira is what's actually released here — the talent's
	// take-home after GigPurse's commission, already smaller than
	// milestone.Amount (see pricing.go). The client's platform fee isn't
	// part of this release; it was already collected as part of funding.
	agreement, err := u.escrowRepo.GetByReference(ctx, milestone.EscrowReference)
	if err != nil {
		return nil, fmt.Errorf("escrow agreement not found: %w", err)
	}
	agreement.PayoutStatus = "PENDING"
	_ = u.escrowRepo.Update(ctx, agreement)

	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: contract.ClientID, Type: "escrow_release", Amount: agreement.AmountNaira,
		Description: fmt.Sprintf("Payment released: %s", milestone.Title), Reference: milestone.EscrowReference,
	})
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: contract.MusicianID, Type: "payment_received", Amount: agreement.AmountNaira,
		Description: fmt.Sprintf("Payment received: %s", milestone.Title), Reference: milestone.EscrowReference,
	})

	milestone.Status = "released"
	// PayPetal debits its wallet and queues the bank transfer asynchronously
	// — the `trustcore.payment.completed` webhook confirms it actually
	// landed, so this notification is deliberately phrased as "initiated,"
	// not "paid."
	u.notify(ctx, contract.MusicianID, "Payment released",
		fmt.Sprintf("Payment for milestone '%s' (%s) has been sent to your bank account.", milestone.Title, formatNaira(agreement.AmountNaira)), contract.ID)

	return milestone, nil
}

// RequestRefund lets the client who funded a milestone pull the escrow back
// before it's released — the single-item, client-initiated counterpart to
// RefundHeldForContract's dispute-driven sweep below.
func (u *milestoneUsecase) RequestRefund(ctx context.Context, clientID, referenceID string) error {
	agreement, err := u.escrowRepo.GetByReference(ctx, referenceID)
	if err != nil {
		return fmt.Errorf("escrow agreement not found: %w", err)
	}
	if agreement.ScopeType != "milestone" {
		return errors.New("this isn't a milestone escrow agreement")
	}
	if agreement.InitiatorUserID != clientID {
		return errors.New("unauthorized: only the client who paid can request this refund")
	}
	if agreement.Status != "ONGOING" || agreement.PayoutStatus != "NONE" || agreement.RefundStatus != "NONE" {
		return errors.New("this payment isn't refundable — it's already been paid out, refunded, or was never confirmed")
	}
	milestone, err := u.milestoneRepo.GetByID(ctx, agreement.ScopeID)
	if err != nil {
		return fmt.Errorf("milestone not found: %w", err)
	}
	if milestone.Status != "funded" {
		return errors.New("this milestone isn't currently funded")
	}

	if err := u.client.RefundTrustCoreAgreement(ctx, referenceID); err != nil {
		return fmt.Errorf("failed to refund: %w", err)
	}
	agreement.RefundStatus = "PENDING"
	if err := u.escrowRepo.Update(ctx, agreement); err != nil {
		return fmt.Errorf("failed to record refund: %w", err)
	}
	// PayPetal's refund returns "the escrowed amount" — agreement.AmountNaira
	// — to the client; GigPurse's platform fee was collected separately as a
	// merchantCharge at funding time and isn't reversed by this call.
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: clientID, Type: "refund", Amount: agreement.AmountNaira,
		Description: fmt.Sprintf("Refund requested: %s", milestone.Title), Reference: referenceID,
	})
	milestone.Status = "refunded"
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return fmt.Errorf("failed to update milestone: %w", err)
	}
	u.notify(ctx, agreement.CounterpartyUserID, "Milestone payment refunded",
		fmt.Sprintf("The client requested a refund for milestone '%s' (%s). The payment has been returned to them.", milestone.Title, formatNaira(agreement.AmountNaira)), milestone.ContractID)

	return nil
}

// RefundHeldForContract refunds every still-`funded` milestone on a
// contract back to the client — the milestone-side half of a dispute
// resolving in the client's favor (see DisputeUsecase.ResolveDispute, which
// also sweeps any job-level escrow the same way). Best-effort per item: one
// failed refund doesn't stop the rest from being attempted.
func (u *milestoneUsecase) RefundHeldForContract(ctx context.Context, contractID string) error {
	milestones, err := u.milestoneRepo.ListByContract(ctx, contractID)
	if err != nil {
		return err
	}
	for _, milestone := range milestones {
		if milestone.Status != "funded" {
			continue
		}
		_ = u.RefundMilestone(ctx, milestone.ID)
	}
	return nil
}

// RefundMilestone is RefundHeldForContract's single-milestone counterpart —
// ungated by design (the caller has already authorized the acting user; see
// RefundHeldForContract's own doc comment for why).
func (u *milestoneUsecase) RefundMilestone(ctx context.Context, milestoneID string) error {
	milestone, err := u.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("milestone not found: %w", err)
	}
	// Accepts "disputed" too — by the time ResolveDispute calls this, a
	// dispute-scoped milestone has already been flipped to "disputed" (see
	// MarkDisputed). Safe only because this method is never reachable from
	// any HTTP route directly — RequestRefund is the client-facing,
	// "funded"-only equivalent, and stays that way so a client can't
	// self-refund a disputed milestone and bypass the moderator.
	if (milestone.Status != "funded" && milestone.Status != "disputed") || milestone.EscrowReference == "" {
		return nil
	}
	// Same claim-before-calling-PayPetal guard as releaseMilestone — no
	// known live trigger for a concurrent double-refund today, but this
	// closes the same class of race rather than leaving it for the next
	// caller who adds one.
	startingStatus := milestone.Status
	won, err := u.milestoneRepo.CompareAndSwapStatus(ctx, milestone.ID, startingStatus, "refunded")
	if err != nil {
		return err
	}
	if !won {
		return nil
	}
	agreement, err := u.escrowRepo.GetByReference(ctx, milestone.EscrowReference)
	if err != nil {
		return err
	}
	if err := u.client.RefundTrustCoreAgreement(ctx, milestone.EscrowReference); err != nil {
		_, _ = u.milestoneRepo.CompareAndSwapStatus(ctx, milestone.ID, "refunded", startingStatus)
		return err
	}
	agreement.RefundStatus = "PENDING"
	_ = u.escrowRepo.Update(ctx, agreement)
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: agreement.InitiatorUserID, Type: "refund", Amount: agreement.AmountNaira,
		Description: fmt.Sprintf("Escrow refunded (dispute resolved): %s", milestone.Title), Reference: milestone.EscrowReference,
	})
	return nil
}

// CancelAccepted cancels a still-`accepted` milestone — ungated (caller
// already authorized), no dispute involved since no money moved yet.
func (u *milestoneUsecase) CancelAccepted(ctx context.Context, milestoneID string) error {
	milestone, err := u.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("milestone not found: %w", err)
	}
	if milestone.Status != "accepted" {
		return fmt.Errorf("milestone is %s, not accepted", milestone.Status)
	}
	milestone.Status = "cancelled"
	return u.milestoneRepo.Update(ctx, milestone)
}

// MarkDisputed locks a `funded` milestone out of Release/RequestRefund while
// the dispute CancelMilestone just opened on it (disputeID) is pending.
// Ungated for the same reason as CancelAccepted.
func (u *milestoneUsecase) MarkDisputed(ctx context.Context, milestoneID, disputeID string) error {
	milestone, err := u.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("milestone not found: %w", err)
	}
	if milestone.Status != "funded" {
		return fmt.Errorf("milestone is %s, not funded", milestone.Status)
	}
	milestone.Status = "disputed"
	milestone.DisputeID = disputeID
	return u.milestoneRepo.Update(ctx, milestone)
}

// CreateSettlementMilestone is how ResolveDispute pays out a partial award
// — see its doc comment on the MilestoneUsecase interface for why this
// reuses the ordinary Fund/FinalizeFund flow instead of a parallel payment
// path. Skips the usual propose/accept dance (status starts at "accepted")
// since a dispute ruling doesn't need the talent's separate buy-in.
func (u *milestoneUsecase) CreateSettlementMilestone(ctx context.Context, contractID, disputeID, resolverID string, amountNaira float64) (*domain.Milestone, error) {
	if amountNaira <= 0 {
		return nil, errors.New("settlement amount must be greater than zero")
	}
	existing, err := u.milestoneRepo.ListByContract(ctx, contractID)
	if err != nil {
		return nil, err
	}
	m := &domain.Milestone{
		ContractID: contractID,
		Title:      "Dispute settlement",
		Amount:     amountNaira,
		Status:     "accepted",
		ProposedBy: resolverID,
		DisputeID:  disputeID,
		Order:      len(existing),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := u.milestoneRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	if contract, err := u.contractRepo.GetByID(ctx, contractID); err == nil {
		u.notify(ctx, contract.ClientID, "Payment required",
			fmt.Sprintf("A moderator ordered a %s payment to settle a dispute — fund it to continue using GigPurse.", formatNaira(amountNaira)), contractID)
	}
	return m, nil
}

// RequestRelease lets the talent ask for a `funded` milestone to be paid
// out — StartAutoReleaseScanner pays it after 48h of client silence.
func (u *milestoneUsecase) RequestRelease(ctx context.Context, contractID, milestoneID, talentID string) error {
	contract, milestone, _, err := u.loadForTransition(ctx, contractID, milestoneID, talentID)
	if err != nil {
		return err
	}
	if talentID != contract.MusicianID {
		return errors.New("unauthorized: only the talent can request a release")
	}
	if milestone.Status != "funded" {
		return errors.New("milestone is not funded yet")
	}
	now := time.Now()
	milestone.ReleaseRequestedAt = &now
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return err
	}
	u.notify(ctx, contract.ClientID, "Release requested",
		fmt.Sprintf("The talent asked you to release '%s' (%s) — it auto-releases in 48h if you don't act.", milestone.Title, formatNaira(milestone.Amount)), contractID)
	return nil
}

func (u *milestoneUsecase) List(ctx context.Context, contractID, requesterID string) ([]*domain.Milestone, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	if _, ok := u.participant(contract, requesterID); !ok {
		return nil, errors.New("unauthorized: not a participant on this contract")
	}
	return u.milestoneRepo.ListByContract(ctx, contractID)
}

// StartReminderScanner polls every checkInterval for milestones stuck in
// "proposed" for at least nudgeAfter since they were last nudged (or since
// they were proposed/countered, if never nudged yet), and re-notifies
// whichever party hasn't responded — repeating every nudgeAfter until they
// do. Runs until ctx is cancelled.
func (u *milestoneUsecase) StartReminderScanner(ctx context.Context, checkInterval, nudgeAfter time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.sendDueReminders(ctx, nudgeAfter)
			}
		}
	}()
}

func (u *milestoneUsecase) sendDueReminders(ctx context.Context, nudgeAfter time.Duration) {
	pending, err := u.milestoneRepo.ListByStatus(ctx, "proposed")
	if err != nil {
		return
	}
	now := time.Now()
	for _, m := range pending {
		baseline := m.UpdatedAt
		if m.LastReminderAt != nil && m.LastReminderAt.After(baseline) {
			baseline = *m.LastReminderAt
		}
		if now.Sub(baseline) < nudgeAfter {
			continue
		}
		contract, err := u.contractRepo.GetByID(ctx, m.ContractID)
		if err != nil {
			continue
		}
		respondent, ok := u.participant(contract, m.ProposedBy)
		if !ok {
			continue
		}
		u.notify(ctx, respondent, "Milestone awaiting your response",
			fmt.Sprintf("You still haven't responded to the milestone '%s' (%s) — open it and accept, reject, or counter.", m.Title, formatNaira(m.Amount)), m.ContractID)

		reminderTime := now
		m.LastReminderAt = &reminderTime
		_ = u.milestoneRepo.Update(ctx, m)
	}
}

// autoReleaseAfter is how long a client has to act (release, refund, or
// dispute) after the talent requests a release before the system releases
// it for them.
const autoReleaseAfter = 48 * time.Hour

// StartAutoReleaseScanner runs in the background for the lifetime of ctx,
// releasing any `funded` milestone whose RequestRelease window has elapsed
// with no client response. Same shape as StartReminderScanner.
func (u *milestoneUsecase) StartAutoReleaseScanner(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.releaseDueMilestones(ctx)
			}
		}
	}()
}

func (u *milestoneUsecase) releaseDueMilestones(ctx context.Context) {
	funded, err := u.milestoneRepo.ListByStatus(ctx, "funded")
	if err != nil {
		return
	}
	now := time.Now()
	for _, m := range funded {
		if m.ReleaseRequestedAt == nil || now.Sub(*m.ReleaseRequestedAt) < autoReleaseAfter {
			continue
		}
		contract, err := u.contractRepo.GetByID(ctx, m.ContractID)
		if err != nil {
			continue
		}
		// Release is normally client-gated; the system is acting on the
		// client's behalf here because their 48h window lapsed, so it calls
		// through with the client's own ID exactly like dispute resolution
		// already does for a full-release ruling.
		if _, err := u.Release(ctx, m.ContractID, m.ID, contract.ClientID); err != nil {
			continue
		}
		u.notify(ctx, contract.ClientID, "Milestone auto-released",
			fmt.Sprintf("'%s' was automatically released to the talent after 48h with no response from you.", m.Title), m.ContractID)
	}
}
