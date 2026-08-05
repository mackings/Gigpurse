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

type contractUsecase struct {
	paypetalDeps
	contractRepo domain.ContractRepository
	jobRepo      domain.JobRepository
	notifRepo    domain.NotificationRepository
	walletRepo   domain.WalletRepository
	escrowRepo   domain.EscrowAgreementRepository
}

func NewContractUsecase(
	contractRepo domain.ContractRepository,
	jobRepo domain.JobRepository,
	notifRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
	walletRepo domain.WalletRepository,
	paypetalClient paypetal.API,
	escrowRepo domain.EscrowAgreementRepository,
) domain.ContractUsecase {
	return &contractUsecase{
		paypetalDeps: paypetalDeps{client: paypetalClient, userRepo: userRepo},
		contractRepo: contractRepo,
		jobRepo:      jobRepo,
		notifRepo:    notifRepo,
		walletRepo:   walletRepo,
		escrowRepo:   escrowRepo,
	}
}

func (u *contractUsecase) GetContract(ctx context.Context, requesterID, requesterRole, id string) (*domain.Contract, error) {
	contract, err := u.contractRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if requesterRole != "admin" && contract.ClientID != requesterID && contract.MusicianID != requesterID {
		return nil, errors.New("unauthorized: contract is not available to this user")
	}
	return contract, nil
}

func (u *contractUsecase) ListUserContracts(ctx context.Context, userID, role string) ([]*domain.Contract, error) {
	return u.contractRepo.ListForUser(ctx, userID, role)
}

func (u *contractUsecase) CompleteContract(ctx context.Context, clientID, contractID string) error {
	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return fmt.Errorf("contract not found: %w", err)
	}

	if contract.ClientID != clientID {
		return errors.New("unauthorized: only the client can mark a contract as completed")
	}

	if contract.Status != "active" {
		return fmt.Errorf("cannot complete contract in status: %s", contract.Status)
	}

	// Update contract
	contract.Status = "completed"
	contract.UpdatedAt = time.Now()
	if err := u.contractRepo.Update(ctx, contract); err != nil {
		return fmt.Errorf("failed to complete contract: %w", err)
	}

	// Update Job
	job, err := u.jobRepo.GetByID(ctx, contract.JobID)
	if err == nil && job != nil {
		job.Status = "completed"
		job.UpdatedAt = time.Now()
		_ = u.jobRepo.Update(ctx, job)
	}
	jobTitle := contract.Title
	if job != nil && job.Title != "" {
		jobTitle = job.Title
	}

	// Release job-level escrow (the whole-budget-up-front case, no
	// milestones) to the musician now that the client has confirmed the
	// work is done — this is the only trigger for that money ever moving;
	// unlike milestones there's no separate "release" button for it.
	// Best-effort: a release failure here shouldn't block marking the
	// contract complete, since the client's own action already succeeded.
	if job != nil && job.EscrowFunded && contract.EscrowReference != "" {
		if err := u.releaseJobEscrow(ctx, contract, job); err != nil {
			log.Printf("contract %s: job escrow release failed: %v", contract.ID, err)
		}
	}

	// Send Notifications
	contractLink := "/contracts/" + contract.ID
	u.notifyAndEmail(ctx, contract.ClientID, "Contract Completed", fmt.Sprintf("The contract '%s' has been marked completed. Please leave a review for the musician.", jobTitle), contractLink)
	u.notifyAndEmail(ctx, contract.MusicianID, "Contract Completed", fmt.Sprintf("Your contract '%s' has been marked completed by the client. Please leave a review for the client.", jobTitle), contractLink)

	return nil
}

// releaseJobEscrow pays out a job-sourced contract's lump-sum escrow to the
// musician — mirrors disputeUsecase.resolveJobEscrow's musician-wins branch
// exactly, just triggered by normal completion instead of a dispute ruling.
func (u *contractUsecase) releaseJobEscrow(ctx context.Context, contract *domain.Contract, job *domain.Job) error {
	agreement, err := u.escrowRepo.GetByReference(ctx, contract.EscrowReference)
	if err != nil {
		return fmt.Errorf("escrow agreement not found: %w", err)
	}
	if agreement.PayoutStatus != "NONE" || agreement.RefundStatus != "NONE" {
		return nil // already settled (e.g. a dispute got there first)
	}

	if err := u.client.CompleteTrustCoreAgreement(ctx, contract.EscrowReference); err != nil {
		return fmt.Errorf("failed to release payment: %w", err)
	}
	agreement.PayoutStatus = "PENDING"
	_ = u.escrowRepo.Update(ctx, agreement)

	if u.walletRepo != nil {
		_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
			UserID: contract.MusicianID, Type: "payment_received", Amount: job.EscrowAmount,
			Description: fmt.Sprintf("Payment released: %s", job.Title), Reference: contract.EscrowReference,
		})
	}

	job.EscrowAmount = 0
	job.EscrowFunded = false
	_ = u.jobRepo.Update(ctx, job)
	return nil
}

// CreateDirectHireRequest lets a client start a direct booking with a
// musician. Only clients initiate — musicians respond via
// RespondToDirectHireRequest/CounterDirectHireRequest on the request the
// client sent, they don't originate one themselves.
func (u *contractUsecase) CreateDirectHireRequest(ctx context.Context, initiatorID, counterpartID string, terms domain.DirectHireTerms) (*domain.DirectHireRequest, error) {
	if initiatorID == "" || counterpartID == "" || terms.Title == "" || terms.Description == "" || terms.Price <= 0 {
		return nil, errors.New("counterpart, title, description, and positive price are required")
	}
	if u.userRepo == nil {
		return nil, errors.New("direct hire requests require user role verification")
	}
	initiator, err := u.userRepo.GetByID(ctx, initiatorID)
	if err != nil {
		return nil, errors.New("initiator not found")
	}
	counterpart, err := u.userRepo.GetByID(ctx, counterpartID)
	if err != nil {
		return nil, errors.New("counterpart not found")
	}
	if initiator.Role != "client" {
		return nil, errors.New("only clients can propose a booking; talent can respond to a booking a client sends")
	}
	if counterpart.Role != "musician" {
		return nil, errors.New("direct hire target must be a musician")
	}
	clientID, musicianID := initiatorID, counterpartID

	now := time.Now()
	req := &domain.DirectHireRequest{
		ClientID:    clientID,
		MusicianID:  musicianID,
		Title:       terms.Title,
		Description: terms.Description,
		Location:    terms.Location,
		EventDate:   terms.EventDate,
		Price:       terms.Price,
		ProposedBy:  initiatorID,
		History: []domain.NegotiationEntry{
			{ProposedBy: initiatorID, Price: terms.Price, Description: terms.Description, Location: terms.Location, EventDate: terms.EventDate, CreatedAt: now},
		},
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := u.contractRepo.CreateDirectHireRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create direct hire request: %w", err)
	}
	u.notifyAndEmail(ctx, counterpartID, "Direct Hire Request", fmt.Sprintf("You received a direct hire request: '%s'.", terms.Title), "/messages?with="+initiatorID+"&booking="+req.ID)
	return req, nil
}

func (u *contractUsecase) ListDirectHireRequests(ctx context.Context, userID, role, status string) ([]*domain.DirectHireRequest, error) {
	return u.contractRepo.ListDirectHireRequestsForUser(ctx, userID, role, status)
}

func (u *contractUsecase) GetDirectHireRequest(ctx context.Context, userID, requestID string) (*domain.DirectHireRequest, error) {
	req, err := u.contractRepo.GetDirectHireRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if userID != req.ClientID && userID != req.MusicianID {
		return nil, errors.New("unauthorized: not a participant on this booking request")
	}
	return req, nil
}

// counterpartOf returns the other participant on the request, given who is
// acting — used both for authorization (only the non-proposer may respond)
// and to know who to notify.
func counterpartOf(req *domain.DirectHireRequest, userID string) (counterpart string, ok bool) {
	switch userID {
	case req.ClientID:
		return req.MusicianID, true
	case req.MusicianID:
		return req.ClientID, true
	default:
		return "", false
	}
}

func (u *contractUsecase) RespondToDirectHireRequest(ctx context.Context, userID, requestID, decision string) (*domain.DirectHireRequest, error) {
	req, err := u.contractRepo.GetDirectHireRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	counterpart, ok := counterpartOf(req, userID)
	if !ok {
		return nil, errors.New("unauthorized: not a participant on this booking request")
	}
	if userID == req.ProposedBy {
		return nil, errors.New("you made the current offer; the other party must respond to it")
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("cannot respond to request in status: %s", req.Status)
	}

	now := time.Now()
	switch decision {
	case "accepted":
		contract := &domain.Contract{
			ClientID:    req.ClientID,
			MusicianID:  req.MusicianID,
			Title:       req.Title,
			Description: req.Description,
			Price:       req.Price,
			Source:      "direct_hire",
			Status:      "active",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := u.contractRepo.Create(ctx, contract); err != nil {
			return nil, fmt.Errorf("failed to create direct hire contract: %w", err)
		}
		req.Status = "accepted"
		req.ContractID = contract.ID
		u.notifyAndEmail(ctx, counterpart, "Booking Accepted", fmt.Sprintf("Your booking '%s' was accepted.", req.Title), "/contracts/"+contract.ID)
	case "declined":
		req.Status = "declined"
		u.notifyAndEmail(ctx, counterpart, "Booking Declined", fmt.Sprintf("Your booking '%s' was declined.", req.Title), "/messages?with="+userID+"&booking="+req.ID)
	default:
		return nil, errors.New("decision must be 'accepted' or 'declined'")
	}
	req.UpdatedAt = now
	if err := u.contractRepo.UpdateDirectHireRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to update direct hire request: %w", err)
	}
	return req, nil
}

func (u *contractUsecase) CounterDirectHireRequest(ctx context.Context, userID, requestID string, terms domain.DirectHireTerms) (*domain.DirectHireRequest, error) {
	if terms.Price <= 0 {
		return nil, errors.New("counter-offer needs a positive price")
	}
	req, err := u.contractRepo.GetDirectHireRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	counterpart, ok := counterpartOf(req, userID)
	if !ok {
		return nil, errors.New("unauthorized: not a participant on this booking request")
	}
	if userID == req.ProposedBy {
		return nil, errors.New("you made the current offer; the other party must respond to it")
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("cannot counter a request in status: %s", req.Status)
	}

	now := time.Now()
	if terms.Title != "" {
		req.Title = terms.Title
	}
	if terms.Description != "" {
		req.Description = terms.Description
	}
	if terms.Location != "" {
		req.Location = terms.Location
	}
	if terms.EventDate != nil {
		req.EventDate = terms.EventDate
	}
	req.Price = terms.Price
	req.ProposedBy = userID
	req.History = append(req.History, domain.NegotiationEntry{
		ProposedBy: userID, Price: req.Price, Description: req.Description, Location: req.Location, EventDate: req.EventDate, CreatedAt: now,
	})
	req.UpdatedAt = now
	if err := u.contractRepo.UpdateDirectHireRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to update direct hire request: %w", err)
	}
	u.notifyAndEmail(ctx, counterpart, "Booking Terms Updated", fmt.Sprintf("New offer for '%s': %s", req.Title, formatNaira(req.Price)), "/messages?with="+userID+"&booking="+req.ID)
	return req, nil
}

func (u *contractUsecase) notifyAndEmail(ctx context.Context, userID, title, message, link string) {
	// Create In-App Notification
	notif := &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Link:      link,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)

	// Mock Email Sending (Email Notification Feature)
	log.Printf("[EMAIL OUTBOX] To User %s: Subject: %s | Message: %s", userID, title, message)
}
