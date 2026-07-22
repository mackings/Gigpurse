package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gigpurse/internal/domain"
)

type disputeUsecase struct {
	disputeRepo      domain.DisputeRepository
	contractRepo     domain.ContractRepository
	notifRepo        domain.NotificationRepository
	chatRepo         domain.ChatRepository
	userRepo         domain.UserRepository
	jobRepo          domain.JobRepository
	walletRepo       domain.WalletRepository
	milestoneUsecase domain.MilestoneUsecase
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
) domain.DisputeUsecase {
	return &disputeUsecase{
		disputeRepo:      disputeRepo,
		contractRepo:     contractRepo,
		notifRepo:        notifRepo,
		chatRepo:         chatRepo,
		userRepo:         userRepo,
		jobRepo:          jobRepo,
		walletRepo:       walletRepo,
		milestoneUsecase: milestoneUsecase,
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

	now := time.Now()
	dispute := &domain.Dispute{
		ContractID: contractID,
		ClientID:   contract.ClientID,
		MusicianID: contract.MusicianID,
		OpenedByID: userID,
		Reason:     reason,
		Status:     "open",
		CreatedAt:  now,
		UpdatedAt:  now,
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

func (u *disputeUsecase) ListUserDisputes(ctx context.Context, userID string) ([]*domain.Dispute, error) {
	return u.disputeRepo.ListForUser(ctx, userID)
}

func (u *disputeUsecase) ListAllDisputes(ctx context.Context, status string) ([]*domain.Dispute, error) {
	return u.disputeRepo.List(ctx, status)
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

// ResolveDispute requires a winner and sweeps any escrow still held against
// the dispute's contract to whichever side won — every still-`funded`
// milestone, plus job-level escrow for a job-sourced contract that never
// used milestones (see jobRepo/walletRepo usage below, which mirrors
// jobUsecase.CloseJob's refund path exactly, just routable to either side).
func (u *disputeUsecase) ResolveDispute(ctx context.Context, resolverID, disputeID, winnerID, resolution string) (*domain.Dispute, error) {
	if disputeID == "" || resolution == "" {
		return nil, errors.New("dispute_id and resolution are required")
	}
	dispute, err := u.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if winnerID != dispute.ClientID && winnerID != dispute.MusicianID {
		return nil, errors.New("winner_id must be one of the two parties on this dispute")
	}

	dispute.Status = "resolved"
	dispute.Resolution = resolution
	dispute.WinnerID = winnerID
	dispute.UpdatedAt = time.Now()
	if err := u.disputeRepo.Update(ctx, dispute); err != nil {
		return nil, fmt.Errorf("failed to resolve dispute: %w", err)
	}

	clientWon := winnerID == dispute.ClientID
	if err := u.milestoneUsecase.RefundHeldForContract(ctx, dispute.ContractID); clientWon && err != nil {
		log.Printf("dispute %s: milestone refund sweep failed: %v", disputeID, err)
	} else if !clientWon {
		// Musician won — any funded-but-unreleased milestone should pay out
		// to them instead of sitting in escrow indefinitely. There's no
		// bulk "release" usecase method (Release is per-milestone and
		// caller-gated to the client), so this walks the list directly.
		if milestones, err := u.milestonesFor(ctx, dispute.ContractID); err == nil {
			for _, m := range milestones {
				if m.Status == "funded" {
					if _, err := u.milestoneUsecase.Release(ctx, dispute.ContractID, m.ID, dispute.ClientID); err != nil {
						log.Printf("dispute %s: milestone %s release failed: %v", disputeID, m.ID, err)
					}
				}
			}
		}
	}

	u.resolveJobEscrow(ctx, dispute, clientWon)

	winnerLabel := "the client"
	if !clientWon {
		winnerLabel = "the talent"
	}
	summary := fmt.Sprintf("%s — ruled in favor of %s. Any held escrow has been settled accordingly.", resolution, winnerLabel)
	u.notify(ctx, dispute.ClientID, "Dispute Resolved", summary, "/messages?dispute="+disputeID)
	u.notify(ctx, dispute.MusicianID, "Dispute Resolved", summary, "/messages?dispute="+disputeID)
	_ = u.chatRepo.SaveMessage(ctx, &domain.ChatMessage{
		DisputeID: disputeID,
		SenderID:  resolverID,
		IsSystem:  true,
		Content:   fmt.Sprintf("Dispute resolved in favor of %s. %s", winnerLabel, resolution),
		Timestamp: time.Now(),
	})

	return dispute, nil
}

func (u *disputeUsecase) milestonesFor(ctx context.Context, contractID string) ([]*domain.Milestone, error) {
	return u.milestoneUsecase.List(ctx, contractID, "")
}

// resolveJobEscrow handles the case where the contract came straight from a
// job whose whole budget was funded up front and never split into
// milestones — mirrors jobUsecase.CloseJob's refund-to-client math, just
// routable to the musician too when they're the winner.
func (u *disputeUsecase) resolveJobEscrow(ctx context.Context, dispute *domain.Dispute, clientWon bool) {
	contract, err := u.contractRepo.GetByID(ctx, dispute.ContractID)
	if err != nil || contract.JobID == "" {
		return
	}
	job, err := u.jobRepo.GetByID(ctx, contract.JobID)
	if err != nil || !job.EscrowFunded || job.EscrowAmount <= 0 {
		return
	}

	payeeID := dispute.MusicianID
	if clientWon {
		payeeID = dispute.ClientID
	}

	clientWallet, err := u.walletRepo.GetOrCreate(ctx, dispute.ClientID)
	if err != nil {
		return
	}
	clientWallet.EscrowBalance -= job.EscrowAmount
	_ = u.walletRepo.Save(ctx, clientWallet)

	if clientWon {
		clientWallet.Balance += job.EscrowAmount
		_ = u.walletRepo.Save(ctx, clientWallet)
	} else {
		musicianWallet, err := u.walletRepo.GetOrCreate(ctx, dispute.MusicianID)
		if err == nil {
			musicianWallet.Balance += job.EscrowAmount
			musicianWallet.TotalEarned += job.EscrowAmount
			_ = u.walletRepo.Save(ctx, musicianWallet)
		}
	}
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: payeeID, Type: "escrow_release", Amount: job.EscrowAmount,
		Description: fmt.Sprintf("Dispute resolved — escrow settled for gig: %s", job.Title),
	})

	job.EscrowAmount = 0
	job.EscrowFunded = false
	_ = u.jobRepo.Update(ctx, job)
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
	log.Printf("[EMAIL OUTBOX] To User %s: Subject: %s | Message: %s", userID, title, message)
}
