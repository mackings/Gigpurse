package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gigpurse/internal/domain"
)

type contractUsecase struct {
	contractRepo domain.ContractRepository
	jobRepo      domain.JobRepository
	notifRepo    domain.NotificationRepository
	userRepo     domain.UserRepository
}

func NewContractUsecase(
	contractRepo domain.ContractRepository,
	jobRepo domain.JobRepository,
	notifRepo domain.NotificationRepository,
	userRepos ...domain.UserRepository,
) domain.ContractUsecase {
	var userRepo domain.UserRepository
	if len(userRepos) > 0 {
		userRepo = userRepos[0]
	}
	return &contractUsecase{
		contractRepo: contractRepo,
		jobRepo:      jobRepo,
		notifRepo:    notifRepo,
		userRepo:     userRepo,
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

	// Send Notifications
	u.notifyAndEmail(ctx, contract.ClientID, "Contract Completed", fmt.Sprintf("The contract '%s' has been marked completed. Please leave a review for the musician.", jobTitle))
	u.notifyAndEmail(ctx, contract.MusicianID, "Contract Completed", fmt.Sprintf("Your contract '%s' has been marked completed by the client. Please leave a review for the client.", jobTitle))

	return nil
}

func (u *contractUsecase) CreateDirectHireRequest(ctx context.Context, clientID, musicianID, title, description string, price float64) (*domain.DirectHireRequest, error) {
	if clientID == "" || musicianID == "" || title == "" || description == "" || price <= 0 {
		return nil, errors.New("client, musician, title, description, and positive price are required")
	}
	if u.userRepo != nil {
		client, err := u.userRepo.GetByID(ctx, clientID)
		if err != nil || client.Role != "client" {
			return nil, errors.New("only clients can create direct hire requests")
		}
		musician, err := u.userRepo.GetByID(ctx, musicianID)
		if err != nil || musician.Role != "musician" {
			return nil, errors.New("direct hire target must be a musician")
		}
	}

	now := time.Now()
	req := &domain.DirectHireRequest{
		ClientID:    clientID,
		MusicianID:  musicianID,
		Title:       title,
		Description: description,
		Price:       price,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := u.contractRepo.CreateDirectHireRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create direct hire request: %w", err)
	}
	u.notifyAndEmail(ctx, musicianID, "Direct Hire Request", fmt.Sprintf("You received a direct hire request: '%s'.", title))
	return req, nil
}

func (u *contractUsecase) ListDirectHireRequests(ctx context.Context, userID, role, status string) ([]*domain.DirectHireRequest, error) {
	return u.contractRepo.ListDirectHireRequestsForUser(ctx, userID, role, status)
}

func (u *contractUsecase) RespondToDirectHireRequest(ctx context.Context, musicianID, requestID, decision string) (*domain.DirectHireRequest, error) {
	req, err := u.contractRepo.GetDirectHireRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.MusicianID != musicianID {
		return nil, errors.New("unauthorized: only the requested musician can respond")
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
		u.notifyAndEmail(ctx, req.ClientID, "Direct Hire Accepted", fmt.Sprintf("Your direct hire request '%s' was accepted.", req.Title))
	case "declined":
		req.Status = "declined"
		u.notifyAndEmail(ctx, req.ClientID, "Direct Hire Declined", fmt.Sprintf("Your direct hire request '%s' was declined.", req.Title))
	default:
		return nil, errors.New("decision must be 'accepted' or 'declined'")
	}
	req.UpdatedAt = now
	if err := u.contractRepo.UpdateDirectHireRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to update direct hire request: %w", err)
	}
	return req, nil
}

func (u *contractUsecase) notifyAndEmail(ctx context.Context, userID, title, message string) {
	// Create In-App Notification
	notif := &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)

	// Mock Email Sending (Email Notification Feature)
	log.Printf("[EMAIL OUTBOX] To User %s: Subject: %s | Message: %s", userID, title, message)
}
