package usecase

import (
	"context"
	"errors"
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
}

func NewPayoutAccountUsecase(client paypetal.API, userRepo domain.UserRepository) PayoutAccountUsecase {
	return &payoutAccountUsecase{paypetalDeps: paypetalDeps{client: client, userRepo: userRepo}}
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
	return user, nil
}
