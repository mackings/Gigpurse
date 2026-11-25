package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

type disputeUsecase struct {
	paypetalDeps
	disputeRepo      domain.DisputeRepository
	contractRepo     domain.ContractRepository
	notifRepo        domain.NotificationRepository
	chatRepo         domain.ChatRepository
	jobRepo          domain.JobRepository
	walletRepo       domain.WalletRepository
	milestoneUsecase domain.MilestoneUsecase
	escrowRepo       domain.EscrowAgreementRepository
}

func NewDisputeUsecase(
	disputeRepo domain.DisputeRepository,
	contractRepo domain.ContractRepository,
	notifRepo domain.NotificationRepository,
	chatRepo domain.ChatRepository,
	userRepo domain.UserRepository,
	jobRepo domain.JobRepository,
	walletRepo domain.WalletRepository,
	milestoneUsecase domain.MilestoneUsecase,
	paypetalClient paypetal.API,
	escrowRepo domain.EscrowAgreementRepository,
) domain.DisputeUsecase {
	return &disputeUsecase{
		paypetalDeps:     paypetalDeps{client: paypetalClient, userRepo: userRepo},
		disputeRepo:      disputeRepo,
		contractRepo:     contractRepo,
		notifRepo:        notifRepo,
		chatRepo:         chatRepo,
		jobRepo:          jobRepo,
		walletRepo:       walletRepo,
		milestoneUsecase: milestoneUsecase,
		escrowRepo:       escrowRepo,
	}
}

func (u *disputeUsecase) OpenDispute(ctx context.Context, userID, contractID, reason string) (*domain.Dispute, error) {
	if contractID == "" || reason == "" {
		return nil, errors.New("contract_id and reason are required")
	}
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	if contract.ClientID != userID && contract.MusicianID != userID {
		return nil, errors.New("unauthorized: only contract participants can open disputes")
	}
	milestones, err := u.milestonesFor(ctx, contractID)
	if err != nil {
		return nil, err
	}
	dispute, err := u.openDispute(ctx, userID, contract, reason, "")
	if err != nil {
		return nil, err
	}
	// Lock everything still in escrow, same as EndContract — without this
	// the client could release or refund it themselves, auto-release could
	// pay it out, and ResolveDispute would find nothing in scope to award.
	for _, m := range milestones {
		if m.Status != "funded" {
			continue
		}
		if err := u.milestoneUsecase.MarkDisputed(ctx, m.ID, dispute.ID); err != nil {
			log.Printf("OpenDispute %s: marking milestone %s disputed failed: %v", contractID, m.ID, err)
		}
	}
	return dispute, nil
}

// openDispute is the shared record-creation step behind both a
// user-initiated OpenDispute and the automatic disputes EndContract/
// CancelMilestone trigger when money's already moved — milestoneID scopes
// resolution to one milestone (set by CancelMilestone) or leaves it empty
// to mean "every funded milestone on the contract" (OpenDispute, EndContract).
func (u *disputeUsecase) openDispute(ctx context.Context, userID string, contract *domain.Contract, reason, milestoneID string) (*domain.Dispute, error) {
	now := time.Now()
	dispute := &domain.Dispute{
		ContractID:  contract.ID,
		ClientID:    contract.ClientID,
		MusicianID:  contract.MusicianID,
		OpenedByID:  userID,
		Reason:      reason,
		Status:      "open",
		MilestoneID: milestoneID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := u.disputeRepo.Create(ctx, dispute); err != nil {
		return nil, fmt.Errorf("failed to create dispute: %w", err)
	}

	// A dispute chat room's opening message is the reason itself, posted as
	// a system note both parties see immediately — they land in a room that
	// already explains why they're there instead of a blank thread.
	_ = u.chatRepo.SaveMessage(ctx, &domain.ChatMessage{
		DisputeID: dispute.ID,
		SenderID:  userID,
		IsSystem:  true,
		Content:   fmt.Sprintf("Dispute opened: %s", reason),
		Timestamp: now,
	})

	u.notify(ctx, contract.ClientID, "Dispute Opened", "A dispute was opened on your contract — you'll be notified once a moderator joins the chat.", "/messages?dispute="+dispute.ID)
	u.notify(ctx, contract.MusicianID, "Dispute Opened", "A dispute was opened on your contract — you'll be notified once a moderator joins the chat.", "/messages?dispute="+dispute.ID)
	return dispute, nil
}

// EndContract lets either party end a contract at any time — see
// CancelMilestone for the single-milestone equivalent. Any milestone still
// just "accepted" (nothing funded) cancels outright; any "funded" one means
// real money is on the line, so the contract goes "disputed" and one
// dispute opens (unscoped — covers every funded milestone) instead.
func (u *disputeUsecase) EndContract(ctx context.Context, userID, contractID string) (*domain.Contract, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	if contract.ClientID != userID && contract.MusicianID != userID {
		return nil, errors.New("unauthorized: not a participant on this contract")
	}
	ender, counterpart, err := u.namesFor(ctx, contract, userID)
	if err != nil {
		return nil, err
	}

	milestones, err := u.milestonesFor(ctx, contractID)
	if err != nil {
		return nil, err
	}
	var fundedIDs []string
	for _, m := range milestones {
		switch m.Status {
		case "accepted":
			if err := u.milestoneUsecase.CancelAccepted(ctx, m.ID); err != nil {
				log.Printf("EndContract %s: cancelling milestone %s failed: %v", contractID, m.ID, err)
			}
		case "funded":
			fundedIDs = append(fundedIDs, m.ID)
		}
	}

	if len(fundedIDs) > 0 {
		contract.Status = "disputed"
		if err := u.contractRepo.Update(ctx, contract); err != nil {
			return nil, fmt.Errorf("failed to update contract: %w", err)
		}
		reason := fmt.Sprintf("%s ended the contract while a milestone was funded — pending resolution.", ender)
		dispute, err := u.openDispute(ctx, userID, contract, reason, "")
		if err != nil {
			return nil, err
		}
		// Mark every funded milestone disputed so ResolveDispute's scope
		// filter (and the frontend's Release/RequestRefund gating) sees the
		// same "disputed" status regardless of whether EndContract or
		// CancelMilestone is what put it under dispute.
		for _, id := range fundedIDs {
			if err := u.milestoneUsecase.MarkDisputed(ctx, id, dispute.ID); err != nil {
				log.Printf("EndContract %s: marking milestone %s disputed failed: %v", contractID, id, err)
			}
		}
		return contract, nil
	}

	contract.Status = "cancelled"
	if err := u.contractRepo.Update(ctx, contract); err != nil {
		return nil, fmt.Errorf("failed to update contract: %w", err)
	}
	u.notify(ctx, u.otherParty(contract, userID), "Contract ended",
		fmt.Sprintf("%s ended the contract '%s'.", ender, contract.Title), "/contracts/"+contractID)
	_ = counterpart
	return contract, nil
}

// CancelMilestone pulls one milestone instead of the whole contract — see
// EndContract for the same accepted-vs-funded split applied to a single item.
func (u *disputeUsecase) CancelMilestone(ctx context.Context, userID, contractID, milestoneID string) (*domain.Milestone, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	if contract.ClientID != userID && contract.MusicianID != userID {
		return nil, errors.New("unauthorized: not a participant on this contract")
	}
	milestones, err := u.milestonesFor(ctx, contractID)
	if err != nil {
		return nil, err
	}
	var milestone *domain.Milestone
	for _, m := range milestones {
		if m.ID == milestoneID {
			milestone = m
			break
		}
	}
	if milestone == nil {
		return nil, errors.New("milestone not found on this contract")
	}

	canceller, _, err := u.namesFor(ctx, contract, userID)
	if err != nil {
		return nil, err
	}

	switch milestone.Status {
	case "accepted":
		if err := u.milestoneUsecase.CancelAccepted(ctx, milestone.ID); err != nil {
			return nil, err
		}
		u.notify(ctx, u.otherParty(contract, userID), "Milestone cancelled",
			fmt.Sprintf("%s cancelled the milestone '%s'.", canceller, milestone.Title), "/contracts/"+contractID)
		milestone.Status = "cancelled"
		return milestone, nil
	case "funded":
		reason := fmt.Sprintf("%s pulled the funded milestone '%s' — pending resolution.", canceller, milestone.Title)
		dispute, err := u.openDispute(ctx, userID, contract, reason, milestone.ID)
		if err != nil {
			return nil, err
		}
		// MarkDisputed first, while the repo's own copy is still genuinely
		// "funded" — the in-memory test repo returns shared pointers, so
		// setting milestone.Status here before MarkDisputed re-fetches
		// would make its own funded-only check see "disputed" already.
		if err := u.milestoneUsecase.MarkDisputed(ctx, milestone.ID, dispute.ID); err != nil {
			log.Printf("CancelMilestone %s: marking milestone disputed failed: %v", milestoneID, err)
		}
		milestone.Status = "disputed"
		return milestone, nil
	default:
		return nil, fmt.Errorf("a %s milestone can't be cancelled this way", milestone.Status)
	}
}

// HasOutstandingSettlement reports whether this client has an unfunded
// "Dispute settlement" milestone waiting anywhere — see ResolveDispute for
// how those get created. Used to block new job posts/hires until they pay
// what a moderator already ordered.
func (u *disputeUsecase) HasOutstandingSettlement(ctx context.Context, clientID string) (bool, error) {
	contracts, err := u.contractRepo.ListForUser(ctx, clientID, "client")
	if err != nil {
		return false, err
	}
	for _, c := range contracts {
		milestones, err := u.milestonesFor(ctx, c.ID)
		if err != nil {
			continue
		}
		for _, m := range milestones {
			if m.AwaitingClientSettlement() {
				return true, nil
			}
		}
	}
	return false, nil
}

// namesFor resolves the acting user's display name and the other party's,
// for the plain-language notifications EndContract/CancelMilestone send.
func (u *disputeUsecase) namesFor(ctx context.Context, contract *domain.Contract, actingUserID string) (actorName, otherName string, err error) {
	actor, err := u.userRepo.GetByID(ctx, actingUserID)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}
	otherID := u.otherParty(contract, actingUserID)
	other, err := u.userRepo.GetByID(ctx, otherID)
	if err != nil {
		return actor.Name, "", nil
	}
	return actor.Name, other.Name, nil
}

func (u *disputeUsecase) otherParty(contract *domain.Contract, userID string) string {
	if userID == contract.ClientID {
		return contract.MusicianID
	}
	return contract.ClientID
}

func (u *disputeUsecase) ListUserDisputes(ctx context.Context, userID string) ([]*domain.Dispute, error) {
	return u.disputeRepo.ListForUser(ctx, userID)
}

func (u *disputeUsecase) ListAllDisputes(ctx context.Context, status string) ([]*domain.Dispute, error) {
	disputes, err := u.disputeRepo.List(ctx, status)
	if err != nil {
		return nil, err
	}
	for _, d := range disputes {
		if contract, err := u.contractRepo.GetByID(ctx, d.ContractID); err == nil {
			d.ContractTitle = contract.Title
		}
	}
	return disputes, nil
}

func (u *disputeUsecase) GetDispute(ctx context.Context, requesterID, disputeID string) (*domain.Dispute, error) {
	dispute, err := u.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if !u.isParticipant(ctx, dispute, requesterID) {
		return nil, errors.New("unauthorized: not a participant on this dispute")
	}
	return dispute, nil
}

// isParticipant covers the two original parties, whoever's currently
// assigned as moderator, and any admin/moderator account (so a second
// moderator/admin can always look in even before formally joining).
func (u *disputeUsecase) isParticipant(ctx context.Context, dispute *domain.Dispute, userID string) bool {
	if userID == dispute.ClientID || userID == dispute.MusicianID || userID == dispute.ModeratorID {
		return true
	}
	if user, err := u.userRepo.GetByID(ctx, userID); err == nil && user != nil {
		return user.Role == "admin" || user.Role == "moderator"
	}
	return false
}

// JoinDispute attaches a moderator/admin to the dispute's chat room — this
// is what unblocks messaging between the two original parties (see
// SendDisputeMessage) and posts the automatic "a moderator has joined"
// notice both parties see right in the thread.
func (u *disputeUsecase) JoinDispute(ctx context.Context, moderatorID, disputeID string) (*domain.Dispute, error) {
	moderator, err := u.userRepo.GetByID(ctx, moderatorID)
	if err != nil {
		return nil, errors.New("moderator not found")
	}
	if moderator.Role != "admin" && moderator.Role != "moderator" {
		return nil, errors.New("unauthorized: only admins or moderators can join a dispute")
	}
	dispute, err := u.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if dispute.ModeratorID == moderatorID {
		return dispute, nil // already joined — idempotent
	}
	dispute.ModeratorID = moderatorID
	dispute.UpdatedAt = time.Now()
	if err := u.disputeRepo.Update(ctx, dispute); err != nil {
		return nil, fmt.Errorf("failed to join dispute: %w", err)
	}

	_ = u.chatRepo.SaveMessage(ctx, &domain.ChatMessage{
		DisputeID: dispute.ID,
		SenderID:  moderatorID,
		IsSystem:  true,
		Content:   fmt.Sprintf("%s has been assigned to this dispute as moderator. Please speak the truth only — screenshots and voice notes are welcome as evidence.", moderator.Name),
		Timestamp: time.Now(),
	})
	u.notify(ctx, dispute.ClientID, "Moderator assigned", "A moderator joined your dispute chat — you can both message now.", "/messages?dispute="+dispute.ID)
	u.notify(ctx, dispute.MusicianID, "Moderator assigned", "A moderator joined your dispute chat — you can both message now.", "/messages?dispute="+dispute.ID)

	return dispute, nil
}

// SendDisputeMessage is the source of truth for the "no talking until a
// moderator joins" rule: the moderator can always post (including the
// system join notice above), but either original party is rejected until
// dispute.ModeratorID is set.
func (u *disputeUsecase) SendDisputeMessage(ctx context.Context, senderID, disputeID, content, attachmentURL, attachmentType, mentionedUserID string) (*domain.ChatMessage, error) {
	if content == "" && attachmentURL == "" {
		return nil, errors.New("a message needs text or an attachment")
	}
	dispute, err := u.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, errors.New("dispute not found")
	}
	if !u.isParticipant(ctx, dispute, senderID) {
		return nil, errors.New("unauthorized: not a participant on this dispute")
	}
	isModerator := senderID == dispute.ModeratorID
	if !isModerator {
		if user, err := u.userRepo.GetByID(ctx, senderID); err == nil && user != nil && (user.Role == "admin" || user.Role == "moderator") {
			isModerator = true
		}
	}
	if !isModerator && dispute.ModeratorID == "" {
		return nil, errors.New("waiting for a moderator to join before you can chat here")
	}
	if attachmentType != "" && attachmentType != "image" && attachmentType != "audio" {
		return nil, errors.New("attachment_type must be image or audio")
	}
	if mentionedUserID != "" && mentionedUserID != dispute.ClientID && mentionedUserID != dispute.MusicianID {
		return nil, errors.New("can only tag the two parties on this dispute")
	}

	msg := &domain.ChatMessage{
		DisputeID:       disputeID,
		SenderID:        senderID,
		Content:         content,
		AttachmentURL:   attachmentURL,
		AttachmentType:  attachmentType,
		MentionedUserID: mentionedUserID,
		Timestamp:       time.Now(),
	}
	if err := u.chatRepo.SaveMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	if mentionedUserID != "" {
		u.notify(ctx, mentionedUserID, "You were tagged in a dispute",
			"The moderator is asking for your response — check the dispute chat.", "/messages?dispute="+disputeID)
	}

	return msg, nil
}

func (u *disputeUsecase) ListDisputeMessages(ctx context.Context, requesterID, disputeID string) ([]*domain.ChatMessage, error) {
	dispute, err := u.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, errors.New("dispute not found")
	}
	if !u.isParticipant(ctx, dispute, requesterID) {
		return nil, errors.New("unauthorized: not a participant on this dispute")
	}
	return u.chatRepo.ListByDispute(ctx, disputeID)
}

// settlementEpsilon absorbs float rounding when comparing naira amounts.
const settlementEpsilon = 0.01

// ResolveDispute rules that the talent gets talentAmountNaira of what's held;
// the client gets the rest back. Scope is dispute.MilestoneID (one milestone)
// if set, else every milestone this dispute locked. Nothing moves until the
// client confirms with the code PayPetal emails them — see
// MilestoneUsecase.BeginVaultSettlement.
func (u *disputeUsecase) ResolveDispute(ctx context.Context, resolverID, disputeID, resolution string, talentAmountNaira float64) (*domain.Dispute, error) {
	if disputeID == "" || resolution == "" {
		return nil, errors.New("dispute_id and resolution are required")
	}
	dispute, err := u.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if dispute.Status != "open" {
		return nil, errors.New("this dispute is already resolved")
	}

	allMilestones, err := u.milestonesFor(ctx, dispute.ContractID)
	if err != nil {
		return nil, err
	}
	var inScope []*domain.Milestone
	held := map[string]float64{} // milestone ID -> the talent's full share
	var totalHeld float64
	for _, m := range allMilestones {
		// OpenDispute/EndContract/CancelMilestone all mark a milestone
		// "disputed" the moment its dispute opens (see MarkDisputed).
		if m.Status != "disputed" || m.EscrowReference == "" {
			continue
		}
		if dispute.MilestoneID != "" && m.ID != dispute.MilestoneID {
			continue
		}
		// A contract can have more than one dispute open at once (a
		// cancelled milestone plus a contract-wide one) — never settle
		// money that a different dispute locked.
		if m.DisputeID != "" && m.DisputeID != dispute.ID {
			continue
		}
		agreement, err := u.escrowRepo.GetByReference(ctx, m.EscrowReference)
		if err != nil || !agreement.IsVault() {
			continue
		}
		// The bound is the talent's take-home, not milestone.Amount — the
		// agreed price also includes GigPurse's commission.
		inScope = append(inScope, m)
		held[m.ID] = agreement.AmountNaira
		totalHeld += agreement.AmountNaira
	}
	if talentAmountNaira < 0 || talentAmountNaira > totalHeld+settlementEpsilon {
		return nil, fmt.Errorf("talent_amount must be between 0 and %.2f (the amount actually held in escrow for this dispute)", totalHeld)
	}

	// Rulings are recorded before the dispute closes, so a failure leaves it
	// open to retry rather than closed with the money still locked.
	remaining := talentAmountNaira
	for _, m := range inScope {
		share := min(remaining, held[m.ID])
		if err := u.milestoneUsecase.BeginVaultSettlement(ctx, m.ID, share); err != nil {
			return nil, fmt.Errorf("couldn't record the ruling on '%s': %w", m.Title, err)
		}
		remaining -= share
	}

	dispute.Status = "resolved"
	dispute.Resolution = resolution
	dispute.TalentAmountNaira = talentAmountNaira
	dispute.UpdatedAt = time.Now()
	if err := u.disputeRepo.Update(ctx, dispute); err != nil {
		return nil, fmt.Errorf("failed to resolve dispute: %w", err)
	}

	summary := resolution
	if len(inScope) > 0 {
		award := "client refunded in full"
		if talentAmountNaira > settlementEpsilon {
			award = "talent awarded " + formatNaira(talentAmountNaira)
		}
		summary = fmt.Sprintf("%s — %s; the client must confirm it with the code PayPetal emails them.", resolution, award)
	}
	u.notify(ctx, dispute.ClientID, "Dispute Resolved", summary, "/messages?dispute="+disputeID)
	u.notify(ctx, dispute.MusicianID, "Dispute Resolved", summary, "/messages?dispute="+disputeID)
	_ = u.chatRepo.SaveMessage(ctx, &domain.ChatMessage{
		DisputeID: disputeID,
		SenderID:  resolverID,
		IsSystem:  true,
		Content:   fmt.Sprintf("Dispute resolved. %s", summary),
		Timestamp: time.Now(),
	})

	return dispute, nil
}

// milestonesFor fetches every milestone on a contract. MilestoneUsecase.List
// gates on the requester being a participant, so this passes the contract's
// own ClientID — always a valid participant on their own contract — rather
// than an empty string, which would fail that check every time (this is
// internal, already-authorized dispute-resolution code, not acting on
// behalf of a specific end user).
func (u *disputeUsecase) milestonesFor(ctx context.Context, contractID string) ([]*domain.Milestone, error) {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}
	return u.milestoneUsecase.List(ctx, contractID, contract.ClientID, "")
}

func (u *disputeUsecase) notify(ctx context.Context, userID, title, message, link string) {
	notif := &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Link:      link,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)
	if user, err := u.userRepo.GetByID(ctx, userID); err == nil && user.Email != "" {
		if err := sendEmailFn(user.Email, title, message); err != nil {
			log.Printf("notify: email to %s failed: %v", user.Email, err)
		}
	}
}
