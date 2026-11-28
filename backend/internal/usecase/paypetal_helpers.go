package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

// ErrPayoutAccountRequired signals the counterparty on an escrow agreement
// hasn't linked a bank account yet — PayPetal's own `complete` precondition
// needs one, so this is checked before an agreement is even created rather
// than surfacing as an opaque PayPetal error later. Exported so the HTTP
// handler layer can match it with errors.Is and respond with a specific
// "payout_account_required" error code instead of a generic 400.
var ErrPayoutAccountRequired = errors.New("payout account required: the other party hasn't added a bank account yet")

// ErrPhoneRequired signals this user has no phone number on file — despite
// PayPetal's own docs listing phoneNumber as optional on customer creation,
// their sandbox rejects a phone-less create with "Phone number is
// required" (confirmed directly against the API, not assumed from docs).
// Checked before ever calling PayPetal, for the same reason as
// ErrPayoutAccountRequired: fail with a specific, actionable error instead
// of an opaque provider error.
var ErrPhoneRequired = errors.New("phone number required: add a phone number before you can send or receive a payment")

// paypetalDeps is embedded by every usecase that needs to talk to PayPetal —
// job, milestone, dispute, and the payout-account usecase itself. Keeping
// ensureCustomer/requirePayoutAccount here (rather than duplicated per
// usecase) means "how a GigPurse user becomes a PayPetal customer" has
// exactly one implementation.
type paypetalDeps struct {
	client   paypetal.API
	userRepo domain.UserRepository
}

// ensureCustomer lazily creates a PayPetal customer record for a user the
// first time they're party to a real money movement — never at signup, so
// users who never transact don't accumulate a PayPetal identity. Persists
// the resulting customerId back onto the user so this only happens once.
//
// A stored ID is checked against PayPetal before being reused: IDs are
// per-environment, so one saved while pointed at sandbox is unknown to live
// and every payment made with it fails. Such a user simply gets a new one.
func (d paypetalDeps) ensureCustomer(ctx context.Context, user *domain.User) (string, error) {
	if user.PayPetalCustomerID != "" {
		exists, err := d.client.CustomerExists(ctx, user.PayPetalCustomerID)
		if err != nil {
			return "", err
		}
		if exists {
			return user.PayPetalCustomerID, nil
		}
		log.Printf("paypetal: stored customer for user %s no longer exists, recreating", user.ID)
	}
	if user.Phone == "" {
		return "", ErrPhoneRequired
	}
	customerID, err := d.client.CreateCustomer(ctx, user.Name, user.Email, user.Phone)
	if err != nil {
		// PayPetal has no lookup-by-email/phone endpoint, so "already
		// exists" (e.g. GigPurse's own record of this customerId was lost —
		// a local DB reset, most commonly — while the same email/phone's
		// PayPetal customer still exists from before) can only be resolved
		// by listing every customer and matching by hand.
		if !isAlreadyExistsError(err) {
			return "", err
		}
		customerID, err = d.findExistingCustomer(ctx, user)
		if err != nil {
			return "", err
		}
	}
	user.PayPetalCustomerID = customerID
	if err := d.userRepo.Update(ctx, user); err != nil {
		return "", err
	}
	return customerID, nil
}

// isRejection reports whether PayPetal definitively refused a request, as
// opposed to failing to process it.
func isRejection(err error) bool {
	var apiErr *paypetal.APIError
	return errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 &&
		apiErr.HTTPStatus != http.StatusUnauthorized && apiErr.HTTPStatus != http.StatusTooManyRequests
}

func isAlreadyExistsError(err error) bool {
	var apiErr *paypetal.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message), "already exist")
}

func (d paypetalDeps) findExistingCustomer(ctx context.Context, user *domain.User) (string, error) {
	customers, err := d.client.ListCustomers(ctx)
	if err != nil {
		return "", err
	}
	// Email is the only identifier safe to auto-recover on: it's specific to
	// this user. A phone-only match can belong to a *different* GigPurse
	// user (two accounts sharing a phone number — PayPetal treats phone as
	// unique merchant-wide, independent of GigPurse's own per-user model),
	// so blindly reusing it would silently merge two distinct identities
	// into one PayPetal customer. Surface that as a clear conflict instead.
	for _, c := range customers {
		if user.Email != "" && strings.EqualFold(c.Email, user.Email) {
			return c.CustomerID, nil
		}
	}
	for _, c := range customers {
		if user.Phone != "" && c.PhoneNumber == user.Phone {
			return "", fmt.Errorf("this phone number is already linked to a different PayPetal customer (%s) — use a different phone number", c.Fullname)
		}
	}
	return "", errors.New("paypetal reported this customer already exists, but it couldn't be found in the customer list to reuse")
}

// requirePayoutAccount gates any flow where user is about to be owed money —
// checked before any is paid in, since a missing account can't be fixed by
// anyone except that user.
func (d paypetalDeps) requirePayoutAccount(user *domain.User) error {
	if user.PayoutAccount == nil {
		return ErrPayoutAccountRequired
	}
	return nil
}

// verifyPayoutAccount checks with PayPetal that the bank account on file can
// still be resolved. Payouts are bank transfers to it, so a dead account has
// to be caught before money is paid in, not after.
func (d paypetalDeps) verifyPayoutAccount(ctx context.Context, user *domain.User) error {
	if err := d.requirePayoutAccount(user); err != nil {
		return err
	}
	_, err := d.client.ValidateBankAccount(ctx, user.PayoutAccount.AccountNumber, user.PayoutAccount.BankCode)
	if isRejection(err) {
		return ErrPayoutAccountRequired
	}
	return err
}
