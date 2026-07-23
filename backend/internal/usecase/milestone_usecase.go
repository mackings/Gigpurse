package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gigpurse/internal/domain"
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
	milestoneRepo domain.MilestoneRepository
	contractRepo  domain.ContractRepository
	walletRepo    domain.WalletRepository
	notifRepo     domain.NotificationRepository
	chatRepo      domain.ChatRepository
	broadcaster   Broadcaster
}

func NewMilestoneUsecase(
	milestoneRepo domain.MilestoneRepository,
	contractRepo domain.ContractRepository,
	walletRepo domain.WalletRepository,
	notifRepo domain.NotificationRepository,
	chatRepo domain.ChatRepository,
	broadcaster Broadcaster,
) domain.MilestoneUsecase {
	return &milestoneUsecase{
		milestoneRepo: milestoneRepo,
		contractRepo:  contractRepo,
		walletRepo:    walletRepo,
		notifRepo:     notifRepo,
		chatRepo:      chatRepo,
		broadcaster:   broadcaster,
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

func (u *milestoneUsecase) Fund(ctx context.Context, contractID, milestoneID, userID string) (*domain.Milestone, error) {
	contract, milestone, _, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return nil, err
	}
	if userID != contract.ClientID {
		return nil, errors.New("unauthorized: only the client can fund a milestone")
	}
	if milestone.Status != "accepted" {
		return nil, errors.New("milestone must be accepted by both parties before it can be funded")
	}

	wallet, err := u.walletRepo.GetOrCreate(ctx, contract.ClientID)
	if err != nil {
		return nil, err
	}
	if milestone.Amount > wallet.Balance {
		return nil, errors.New("insufficient wallet balance to fund this milestone")
	}
	wallet.Balance -= milestone.Amount
	wallet.EscrowBalance += milestone.Amount
	wallet.TotalSpent += milestone.Amount
	if err := u.walletRepo.Save(ctx, wallet); err != nil {
		return nil, err
	}
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: contract.ClientID, Type: "escrow_hold", Amount: milestone.Amount,
		Description: fmt.Sprintf("Escrow funded: %s", milestone.Title),
	})

	milestone.Status = "funded"
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return nil, err
	}
	u.notify(ctx, contract.MusicianID, "Escrow funded",
		fmt.Sprintf("Escrow funded for milestone '%s' (%s).", milestone.Title, formatNaira(milestone.Amount)), contractID)

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

	clientWallet, err := u.walletRepo.GetOrCreate(ctx, contract.ClientID)
	if err != nil {
		return nil, err
	}
	clientWallet.EscrowBalance -= milestone.Amount
	if err := u.walletRepo.Save(ctx, clientWallet); err != nil {
		return nil, err
	}

	musicianWallet, err := u.walletRepo.GetOrCreate(ctx, contract.MusicianID)
	if err != nil {
		return nil, err
	}
	musicianWallet.Balance += milestone.Amount
	musicianWallet.TotalEarned += milestone.Amount
	if err := u.walletRepo.Save(ctx, musicianWallet); err != nil {
		return nil, err
	}

	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: contract.ClientID, Type: "escrow_release", Amount: milestone.Amount,
		Description: fmt.Sprintf("Payment released: %s", milestone.Title),
	})
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: contract.MusicianID, Type: "payment_received", Amount: milestone.Amount,
		Description: fmt.Sprintf("Payment received: %s", milestone.Title),
	})

	milestone.Status = "released"
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return nil, err
	}
	u.notify(ctx, contract.MusicianID, "Payment released",
		fmt.Sprintf("Payment released for milestone '%s' (%s).", milestone.Title, formatNaira(milestone.Amount)), contractID)

	return milestone, nil
}

// RefundHeldForContract moves every still-`funded` milestone on a contract
// back to the client's wallet balance — the milestone-side half of a
// dispute resolving in the client's favor (see DisputeUsecase.ResolveDispute,
// which also sweeps any job-level escrow the same way).
func (u *milestoneUsecase) RefundHeldForContract(ctx context.Context, contractID string) error {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}
	milestones, err := u.milestoneRepo.ListByContract(ctx, contractID)
	if err != nil {
		return err
	}

	clientWallet, err := u.walletRepo.GetOrCreate(ctx, contract.ClientID)
	if err != nil {
		return err
	}
	refunded := false
	for _, milestone := range milestones {
		if milestone.Status != "funded" {
			continue
		}
		clientWallet.EscrowBalance -= milestone.Amount
		clientWallet.Balance += milestone.Amount
		refunded = true
		_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
			UserID: contract.ClientID, Type: "escrow_release", Amount: milestone.Amount,
			Description: fmt.Sprintf("Escrow refunded (dispute resolved): %s", milestone.Title),
		})
		milestone.Status = "refunded"
		_ = u.milestoneRepo.Update(ctx, milestone)
	}
	if refunded {
		if err := u.walletRepo.Save(ctx, clientWallet); err != nil {
			return err
		}
	}
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
