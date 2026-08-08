package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

// PayoutAccountUsecase manages the bank account a user is paid escrow
// releases/refunds to. Kept separate from UserUsecase since it needs the
// PayPetal dependency and belongs to the payment domain, not the
// profile-editing domain.
type PayoutAccountUsecase interface {
	ListBanks(ctx context.Context) ([]paypetal.Bank, error)
	// Validate resolves an account number to its registered holder name —
	// the frontend must show this back to the user for confirmation before
	// calling Link, since a payout to the wrong account can't be reversed.
	Validate(ctx context.Context, bankCode, accountNumber string) (accountName string, err error)
	Link(ctx context.Context, userID, bankCode, bankName, accountNumber string) (*domain.User, error)
}

type payoutAccountUsecase struct {
	paypetalDeps
	jobRepo       domain.JobRepository
	contractRepo  domain.ContractRepository
	milestoneRepo domain.MilestoneRepository
	notifRepo     domain.NotificationRepository
}

func NewPayoutAccountUsecase(
	client paypetal.API,
	userRepo domain.UserRepository,
	jobRepo domain.JobRepository,
	contractRepo domain.ContractRepository,
	milestoneRepo domain.MilestoneRepository,
	notifRepo domain.NotificationRepository,
) PayoutAccountUsecase {
	return &payoutAccountUsecase{
		paypetalDeps:  paypetalDeps{client: client, userRepo: userRepo},
		jobRepo:       jobRepo,
		contractRepo:  contractRepo,
		milestoneRepo: milestoneRepo,
		notifRepo:     notifRepo,
	}
}

func (u *payoutAccountUsecase) ListBanks(ctx context.Context) ([]paypetal.Bank, error) {
	return u.client.ListBanks(ctx)
}

func (u *payoutAccountUsecase) Validate(ctx context.Context, bankCode, accountNumber string) (string, error) {
	if bankCode == "" || accountNumber == "" {
		return "", errors.New("bank_code and account_number are required")
	}
	return u.client.ValidateBankAccount(ctx, accountNumber, bankCode)
}

func (u *payoutAccountUsecase) Link(ctx context.Context, userID, bankCode, bankName, accountNumber string) (*domain.User, error) {
	if bankCode == "" || bankName == "" || accountNumber == "" {
		return nil, errors.New("bank_code, bank_name, and account_number are required")
	}
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	hadNoAccountBefore := user.PayoutAccount == nil

	// Re-resolve the account name server-side rather than trusting whatever
	// the client echoes back from the earlier Validate call — the
	// confirmation UI is a UX courtesy, not the authorization boundary.
	accountName, err := u.client.ValidateBankAccount(ctx, accountNumber, bankCode)
	if err != nil {
		return nil, err
	}

	// user.Phone is required (collected at signup) — ensureCustomer returns
	// ErrPhoneRequired if it's somehow still missing (an account predating
	// signup requiring a phone), surfaced by the handler as a specific
	// "phone_required" error.
	customerID, err := u.ensureCustomer(ctx, user)
	if err != nil {
		return nil, err
	}

	if err := u.client.LinkPayoutAccount(ctx, customerID, accountNumber, bankCode); err != nil {
		return nil, err
	}

	user.PayoutAccount = &domain.PayoutAccount{
		BankCode:      bankCode,
		BankName:      bankName,
		AccountNumber: accountNumber,
		AccountName:   accountName,
		LinkedAt:      time.Now(),
	}
	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Only worth telling anyone about the first time — every InitiateHire/
	// Fund attempt that hit requirePayoutAccount before this failed silently
	// from the client's point of view (no record kept of who tried), so this
	// tells every client who's currently blocked on this specific musician
	// rather than just whoever happens to retry on their own.
	if hadNoAccountBefore {
		u.notifyUnblockedClients(ctx, user)
	}

	return user, nil
}

func (u *payoutAccountUsecase) notifyUnblockedClients(ctx context.Context, musician *domain.User) {
	if apps, err := u.jobRepo.ListApplicationsByMusician(ctx, musician.ID); err == nil {
		for _, app := range apps {
			if app.Status != "pending" && app.Status != "shortlisted" {
				continue
			}
			job, err := u.jobRepo.GetByID(ctx, app.JobID)
			if err != nil || job.Status != "open" {
				continue
			}
			u.notify(ctx, job.ClientID, "Talent ready to be hired",
				fmt.Sprintf("%s added a payout account and can now be hired for '%s'.", musician.Name, job.Title))
		}
	}

	if contracts, err := u.contractRepo.ListForUser(ctx, musician.ID, "musician"); err == nil {
		for _, c := range contracts {
			if c.Status != "active" {
				continue
			}
			milestones, err := u.milestoneRepo.ListByContract(ctx, c.ID)
			if err != nil {
				continue
			}
			for _, m := range milestones {
				if m.Status != "accepted" {
					continue
				}
				u.notify(ctx, c.ClientID, "Talent ready to be paid",
					fmt.Sprintf("%s added a payout account — you can now fund milestone '%s'.", musician.Name, m.Title))
			}
		}
	}
}

func (u *payoutAccountUsecase) notify(ctx context.Context, userID, title, message string) {
	_ = u.notifRepo.Create(ctx, &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	})
}
