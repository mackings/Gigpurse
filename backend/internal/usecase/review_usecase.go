package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gigpurse/internal/domain"
)

type reviewUsecase struct {
	reviewRepo   domain.ReviewRepository
	contractRepo domain.ContractRepository
	notifRepo    domain.NotificationRepository
}

func NewReviewUsecase(
	reviewRepo domain.ReviewRepository,
	contractRepo domain.ContractRepository,
	notifRepo domain.NotificationRepository,
) domain.ReviewUsecase {
	return &reviewUsecase{
		reviewRepo:   reviewRepo,
		contractRepo: contractRepo,
		notifRepo:    notifRepo,
	}
}

func (u *reviewUsecase) SubmitReview(ctx context.Context, reviewerID, contractID string, rating int, comment string) (*domain.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	contract, err := u.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("contract not found: %w", err)
	}

	if contract.Status != "completed" {
		return nil, errors.New("reviews can only be submitted for completed contracts")
	}

	// Verify reviewer is part of the contract and determine reviewee
	var revieweeID string
	if contract.ClientID == reviewerID {
		revieweeID = contract.MusicianID
	} else if contract.MusicianID == reviewerID {
		revieweeID = contract.ClientID
	} else {
		return nil, errors.New("unauthorized: reviewer must be a participant of the contract")
	}

	// Check for duplicate review
	existing, err := u.reviewRepo.GetByContractAndReviewer(ctx, contractID, reviewerID)
	if err == nil && existing != nil {
		return nil, errors.New("you have already reviewed the other participant for this contract")
	}

	review := &domain.Review{
		ContractID: contractID,
		JobID:      contract.JobID, // empty for direct-hire-sourced contracts
		ReviewerID: reviewerID,
		RevieweeID: revieweeID,
		Rating:     rating,
		Comment:    comment,
		CreatedAt:  time.Now(),
	}

	if err := u.reviewRepo.Create(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to save review: %w", err)
	}

	// In-app alert and email dispatch
	subject := contract.Title
	if subject == "" {
		subject = "your recent contract"
	}
	msg := fmt.Sprintf("You received a new rating of %d stars for '%s'. Review: '%s'", rating, subject, comment)
	notif := &domain.Notification{
		UserID:    revieweeID,
		Title:     "New Review Received",
		Message:   msg,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)
	log.Printf("[EMAIL OUTBOX] To User %s: Subject: New Review | Message: %s", revieweeID, msg)

	return review, nil
}

func (u *reviewUsecase) GetUserReviews(ctx context.Context, userID string) ([]*domain.Review, error) {
	return u.reviewRepo.ListByReviewee(ctx, userID)
}

func (u *reviewUsecase) GetAverageRating(ctx context.Context, userID string) (float64, error) {
	reviews, err := u.reviewRepo.ListByReviewee(ctx, userID)
	if err != nil {
		return 0, err
	}
	if len(reviews) == 0 {
		return 0, nil
	}

	var sum int
	for _, r := range reviews {
		sum += r.Rating
	}
	return float64(sum) / float64(len(reviews)), nil
}
