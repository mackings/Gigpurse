package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gigpurse/internal/domain"
)

type payoutUsecase struct {
	payoutRepo domain.PayoutRepository
	escrowRepo domain.EscrowAgreementRepository
	walletRepo domain.WalletRepository
	userRepo   domain.UserRepository
	notifRepo  domain.NotificationRepository
}

func NewPayoutUsecase(payoutRepo domain.PayoutRepository, escrowRepo domain.EscrowAgreementRepository, walletRepo domain.WalletRepository, userRepo domain.UserRepository, notifRepo domain.NotificationRepository) domain.PayoutUsecase {
	return &payoutUsecase{payoutRepo: payoutRepo, escrowRepo: escrowRepo, walletRepo: walletRepo, userRepo: userRepo, notifRepo: notifRepo}
}

func (u *payoutUsecase) List(ctx context.Context, status string) ([]*domain.Payout, error) {
	switch status {
	case "":
		status = "due"
	case "pending", "due", "paid":
	default:
		return nil, errors.New("status must be pending, due, or paid")
	}
	payouts, err := u.payoutRepo.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	for _, p := range payouts {
		if user, err := u.userRepo.GetByID(ctx, p.UserID); err == nil {
			p.UserName = user.Name
			p.Account = user.PayoutAccount
		}
	}
	return payouts, nil
}

func (u *payoutUsecase) MarkPaid(ctx context.Context, adminID, payoutID, note string) (*domain.Payout, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return nil, errors.New("enter the bank transfer reference")
	}
	p, err := u.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, errors.New("payout not found")
	}
	won, err := u.payoutRepo.CompareAndSwapStatus(ctx, payoutID, "due", "paid")
	if err != nil {
		return nil, err
	}
	if !won {
		return nil, fmt.Errorf("this payout is %s, not due", p.Status)
	}
	now := time.Now()
	p.Status, p.Note, p.PaidBy, p.PaidAt = "paid", note, adminID, &now
	if err := u.payoutRepo.Update(ctx, p); err != nil {
		return nil, err
	}

	txType, title, message := "payment_received", "Payout sent",
		fmt.Sprintf("%s for '%s' has been sent to your bank account.", formatNaira(p.Amount), p.Title)
	description := fmt.Sprintf("Payment received: %s", p.Title)
	if p.Kind == domain.PayoutRefund {
		txType, title = "refund", "Refund sent"
		message = fmt.Sprintf("Your refund of %s for '%s' has been sent to your bank account.", formatNaira(p.Amount), p.Title)
		description = fmt.Sprintf("Refund: %s", p.Title)
	}
	if err := u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: p.UserID, Type: txType, Amount: p.Amount, Description: description, Reference: p.ReferenceID,
	}); err != nil {
		log.Printf("payout %s: recording transaction failed: %v", p.ID, err)
	}
	if agreement, err := u.escrowRepo.GetByReference(ctx, p.ReferenceID); err == nil {
		if p.Kind == domain.PayoutRefund {
			agreement.RefundStatus = "COMPLETED"
		} else {
			agreement.PayoutStatus = "COMPLETED"
		}
		_ = u.escrowRepo.Update(ctx, agreement)
	}
	_ = u.notifRepo.Create(ctx, &domain.Notification{UserID: p.UserID, Title: title, Message: message, ContractID: p.ContractID, CreatedAt: now})
	if user, err := u.userRepo.GetByID(ctx, p.UserID); err == nil && user.Email != "" {
		if err := sendEmailFn(user.Email, title, message); err != nil {
			log.Printf("payout %s: email failed: %v", p.ID, err)
		}
	}
	return p, nil
}
