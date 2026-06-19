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
	disputeRepo  domain.DisputeRepository
	contractRepo domain.ContractRepository
	notifRepo    domain.NotificationRepository
}

func NewDisputeUsecase(
	disputeRepo domain.DisputeRepository,
	contractRepo domain.ContractRepository,
	notifRepo domain.NotificationRepository,
) domain.DisputeUsecase {
	return &disputeUsecase{
		disputeRepo:  disputeRepo,
		contractRepo: contractRepo,
		notifRepo:    notifRepo,
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
	u.notify(ctx, contract.ClientID, "Dispute Opened", "A dispute was opened on your contract.")
	u.notify(ctx, contract.MusicianID, "Dispute Opened", "A dispute was opened on your contract.")
	return dispute, nil
}

func (u *disputeUsecase) ListUserDisputes(ctx context.Context, userID string) ([]*domain.Dispute, error) {
	return u.disputeRepo.ListForUser(ctx, userID)
}

func (u *disputeUsecase) ListAllDisputes(ctx context.Context, status string) ([]*domain.Dispute, error) {
	return u.disputeRepo.List(ctx, status)
}

func (u *disputeUsecase) ResolveDispute(ctx context.Context, disputeID, resolution string) (*domain.Dispute, error) {
	if disputeID == "" || resolution == "" {
		return nil, errors.New("dispute_id and resolution are required")
	}
	dispute, err := u.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	dispute.Status = "resolved"
	dispute.Resolution = resolution
	dispute.UpdatedAt = time.Now()
	if err := u.disputeRepo.Update(ctx, dispute); err != nil {
		return nil, fmt.Errorf("failed to resolve dispute: %w", err)
	}

	contract, err := u.contractRepo.GetByID(ctx, dispute.ContractID)
	if err == nil && contract != nil {
		u.notify(ctx, contract.ClientID, "Dispute Resolved", resolution)
		u.notify(ctx, contract.MusicianID, "Dispute Resolved", resolution)
	}
	return dispute, nil
}

func (u *disputeUsecase) notify(ctx context.Context, userID, title, message string) {
	notif := &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)
	log.Printf("[EMAIL OUTBOX] To User %s: Subject: %s | Message: %s", userID, title, message)
}
