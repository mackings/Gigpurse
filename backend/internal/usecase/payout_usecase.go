package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

type payoutUsecase struct {
	paypetalDeps
	payoutRepo domain.PayoutRepository
	escrowRepo domain.EscrowAgreementRepository
	walletRepo domain.WalletRepository
	notifRepo  domain.NotificationRepository

	// Automatic payouts — off when subWalletRef is empty. See auto_payout.go.
	subWalletRef string
	redirectURL  string
	// pass serialises everything that moves an automatic payout, so two
	// passes (or a pass and an admin action) never act on one at once.
	pass        sync.Mutex
	walletMu    sync.Mutex
	subWalletID string
	balance     float64
	balanceAt   time.Time
}

// NewPayoutUsecase: subWalletRef names the PayPetal sub-wallet automatic
// payouts are paid from; leave it empty to pay everything by hand.
func NewPayoutUsecase(payoutRepo domain.PayoutRepository, escrowRepo domain.EscrowAgreementRepository, walletRepo domain.WalletRepository, userRepo domain.UserRepository, notifRepo domain.NotificationRepository, client paypetal.API, subWalletRef, frontendBaseURL string) domain.PayoutUsecase {
	return &payoutUsecase{
		paypetalDeps: paypetalDeps{client: client, userRepo: userRepo},
		payoutRepo:   payoutRepo, escrowRepo: escrowRepo, walletRepo: walletRepo, notifRepo: notifRepo,
		subWalletRef: strings.TrimSpace(subWalletRef),
		redirectURL:  strings.TrimRight(frontendBaseURL, "/") + "/wallet",
	}
}

func (u *payoutUsecase) List(ctx context.Context, status string) ([]*domain.Payout, error) {
	switch status {
	case "":
		status = "due"
	case "pending", "due", "sending", "stopping", "paid":
	default:
		return nil, errors.New("status must be pending, due, sending, stopping, or paid")
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
	if p.Status != "due" {
		if p.Status == "sending" || p.Status == "stopping" {
			return nil, errors.New("PayPetal is handling this payout — stop the automatic payout first")
		}
		return nil, fmt.Errorf("this payout is %s, not due", p.Status)
	}
	if !u.markPaid(ctx, p, "due", note, adminID) {
		return nil, errors.New("this payout changed meanwhile — refresh and try again")
	}
	return p, nil
}

// markPaid records p as paid, if it's still in status from, and tells the
// payee. Reports whether this call is what recorded it.
func (u *payoutUsecase) markPaid(ctx context.Context, p *domain.Payout, from, note, paidBy string) bool {
	now := time.Now()
	p.Status, p.Note, p.PaidBy, p.PaidAt, p.AutoError, p.RetryAt, p.WaitingOn = "paid", note, paidBy, &now, "", nil, ""
	won, err := u.payoutRepo.UpdateIfStatus(ctx, p, from)
	if err != nil || !won {
		if err != nil {
			log.Printf("payout %s: recording paid failed: %v", p.ID, err)
		}
		return false
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
	u.tell(ctx, p, title, message)
	return true
}

func (u *payoutUsecase) tell(ctx context.Context, p *domain.Payout, title, message string) {
	_ = u.notifRepo.Create(ctx, &domain.Notification{UserID: p.UserID, Title: title, Message: message, ContractID: p.ContractID, CreatedAt: time.Now()})
	if user, err := u.userRepo.GetByID(ctx, p.UserID); err == nil && user.Email != "" {
		if err := sendEmailFn(user.Email, title, message); err != nil {
			log.Printf("payout %s: email failed: %v", p.ID, err)
		}
	}
}
