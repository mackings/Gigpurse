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
	payouts         domain.PayoutRepository
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
	payouts domain.PayoutRepository,
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
		payouts:         payouts,
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

// Fund starts a TrustVault payment for an accepted milestone and returns a
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
	payoutAccountMissing := func(err error) (string, string, error) {
		u.notify(ctx, musician.ID, "Add your bank account to get paid",
			fmt.Sprintf("A milestone payment for '%s' is waiting on you to add a payout account.", milestone.Title), contractID)
		return "", "", err
	}
	// Checked before any money can be paid in: without the talent's account
	// it can't be paid out, without the client's it can't be refunded.
	if err := u.verifyPayoutAccount(ctx, musician); err != nil {
		if errors.Is(err, ErrPayoutAccountRequired) {
			return payoutAccountMissing(err)
		}
		return "", "", fmt.Errorf("failed to check the talent's bank account: %w", err)
	}
	if err := u.verifyPayoutAccount(ctx, client); err != nil {
		if errors.Is(err, ErrPayoutAccountRequired) {
			return "", "", errors.New("add a working bank account on the Wallet page before paying — it's where a refund would be sent")
		}
		return "", "", fmt.Errorf("failed to check your bank account: %w", err)
	}
	clientCustomerID, err := u.ensureCustomer(ctx, client)
	if err != nil {
		return "", "", fmt.Errorf("failed to register client with payment provider: %w", err)
	}

	// The client pays the agreed price plus GigPurse's service fee into
	// TrustVault; the talent's share is paid out of that after release, and
	// GigPurse keeps the rest. See pricing.go for the split.
	//
	// A legacy "Dispute settlement" milestone (DisputeID set) is fee-free.
	talentAmount := TalentTakeHome(milestone.Amount)
	platformFee := PlatformFee(milestone.Amount)
	if milestone.DisputeID != "" {
		talentAmount = milestone.Amount
		platformFee = 0
	}

	// PayPetal's hosted checkout prepends its own "MCH-" prefix before
	// generating a bank-transfer virtual account, and rejects anything past
	// 35 chars at that point (confirmed live: "milestone:" + a 24-char
	// Mongo ID is 34 chars, +4 for "MCH-" = 38, over the limit — every
	// bank-transfer milestone payment 400'd on PayPetal's /pay/create-account
	// step). Keep this prefix short enough to leave headroom.
	//
	// Letters and digits only. Earlier references were "ms:<id>", and live
	// bank-transfer payments made against them were never matched by
	// PayPetal; every reference in their docs sticks to letters, digits and
	// dashes.
	reference := "ms" + milestoneID
	// PayPetal appends "?status=...&txnId=..." to this URL itself, blindly
	// (no check for an existing "?") — so reference must ride in the path,
	// never a query string, or we get a broken double-"?" redirect.
	redirectURL := u.frontendBaseURL + "/contracts/pending/" + url.PathEscape(reference)

	result, err := u.client.CreateVaultAgreement(ctx, paypetal.CreateVaultInput{
		ReferenceID: reference,
		CustomerID:  clientCustomerID,
		Currency:    "NGN",
		Amount:      paypetal.FormatAmount(talentAmount + platformFee),
		Description: fmt.Sprintf("GigPurse milestone: %s", milestone.Title),
		RedirectURL: redirectURL,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to start payment: %w", err)
	}

	agreement := &domain.EscrowAgreement{
		ReferenceID:         reference,
		ScopeType:           "milestone",
		ScopeID:             milestoneID,
		ContractID:          contractID,
		InitiatorCustomerID: clientCustomerID,
		InitiatorUserID:     contract.ClientID,
		CounterpartyUserID:  contract.MusicianID,
		AmountNaira:         talentAmount,
		PlatformFeeNaira:    platformFee,
		Provider:            domain.ProviderVault,
		PayPetalReference:   result.Reference,
		Status:              "PENDING",
		PayoutStatus:        "NONE",
		RefundStatus:        "NONE",
	}
	// Reopening an unpaid checkout: PayPetal hands back the same pending
	// transaction for the same reference, so refresh the record already on
	// file instead of adding a second one for one payment.
	if existing, err := u.escrowRepo.GetByReference(ctx, reference); err == nil && existing.Status == "PENDING" {
		agreement.ID = existing.ID
		agreement.CreatedAt = existing.CreatedAt
		if err := u.escrowRepo.Update(ctx, agreement); err != nil {
			return "", "", fmt.Errorf("failed to record escrow agreement: %w", err)
		}
	} else if err := u.escrowRepo.Create(ctx, agreement); err != nil {
		return "", "", fmt.Errorf("failed to record escrow agreement: %w", err)
	}

	milestone.EscrowReference = reference
	milestone.EscrowProvider = domain.ProviderVault
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return "", "", err
	}

	return result.PaymentURL, reference, nil
}

// FinalizeFund confirms a milestone payment and flips it to "funded" — the
// step after Fund. Called from the webhook handler, the reconciler and a
// frontend poll; idempotent against all of them firing.
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
		return milestone, nil // already finalized
	}

	if !agreement.IsVault() {
		return nil, errLegacyPayment
	}
	state, err := u.client.GetVaultAgreement(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm payment: %w", err)
	}
	if !state.Paid() {
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

	agreement.Status = "ONGOING"
	_ = u.escrowRepo.Update(ctx, agreement)

	// The client's actual charge is amount + platform fee — milestone.Amount
	// is just the agreed price, not what left their payment method.
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: agreement.InitiatorUserID, Type: "escrow_hold", Amount: agreement.AmountNaira + agreement.PlatformFeeNaira,
		Description: fmt.Sprintf("Escrow funded: %s", milestone.Title), Reference: reference,
	})
	u.notify(ctx, agreement.CounterpartyUserID, "Escrow funded",
		fmt.Sprintf("Escrow funded for milestone '%s' (%s).", milestone.Title, formatNaira(milestone.Amount)), milestone.ContractID)

	return milestone, nil
}

// RequestRefund lets the client who funded a milestone pull the escrow back
// before it's released, with the code PayPetal emails them. The money lands
// in GigPurse's wallet and is queued for an admin to send on.
func (u *milestoneUsecase) RequestRefund(ctx context.Context, clientID, referenceID, code string) error {
	agreement, milestone, err := u.vaultAgreementFor(ctx, clientID, referenceID)
	if err != nil {
		return err
	}
	switch milestone.Status {
	case "funded":
		_, err = u.settleVault(ctx, agreement, milestone, 0, code)
		return err
	case "disputed":
		return errors.New("this payment is under dispute — a moderator has to settle it")
	default:
		return errors.New("this milestone isn't currently funded")
	}
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

// MarkDisputed locks a `funded` milestone out of Release/RequestRefund/
// auto-release while the dispute just opened on it (disputeID) is pending.
// Ungated for the same reason as CancelAccepted.
func (u *milestoneUsecase) MarkDisputed(ctx context.Context, milestoneID, disputeID string) error {
	milestone, err := u.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("milestone not found: %w", err)
	}
	// Claimed atomically: a Release/refund/auto-release that already claimed
	// this milestone must win outright, not have its "released" overwritten
	// back to "disputed" after the money has gone.
	won, err := u.milestoneRepo.CompareAndSwapStatus(ctx, milestoneID, "funded", "disputed")
	if err != nil {
		return err
	}
	if !won {
		return fmt.Errorf("milestone is %s, not funded", milestone.Status)
	}
	milestone.Status = "disputed"
	milestone.DisputeID = disputeID
	return u.milestoneRepo.Update(ctx, milestone)
}

// RequestRelease lets the talent ask for a `funded` milestone to be paid out.
// Only the client can release it, so StartReleaseNudgeScanner keeps reminding
// them once nudgeReleaseAfter has passed.
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
		fmt.Sprintf("The talent asked you to release '%s' (%s) — open the contract and enter the code PayPetal emails you.", milestone.Title, formatNaira(milestone.Amount)), contractID)
	return nil
}

func (u *milestoneUsecase) List(ctx context.Context, contractID, requesterID, requesterRole string) ([]*domain.Milestone, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	_, isParticipant := u.participant(contract, requesterID)
	isStaff := requesterRole == "admin" || requesterRole == "moderator"
	if !isParticipant && !isStaff {
		return nil, errors.New("unauthorized: not a participant on this contract")
	}
	return u.milestoneRepo.ListByContract(ctx, contractID)
}

// StartReminderScanner polls every checkInterval for milestones stuck in
// "proposed": whichever party hasn't responded is nudged once nudgeAfter has
// passed since the offer, then at most once every repeatAfter until they do.
// Runs until ctx is cancelled.
func (u *milestoneUsecase) StartReminderScanner(ctx context.Context, checkInterval, nudgeAfter, repeatAfter time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.sendDueReminders(ctx, nudgeAfter, repeatAfter)
			}
		}
	}()
}

func (u *milestoneUsecase) sendDueReminders(ctx context.Context, nudgeAfter, repeatAfter time.Duration) {
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
		wait := nudgeAfter
		if m.LastReminderAt != nil {
			wait = repeatAfter
		}
		if now.Sub(baseline) < wait {
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
