package usecase

import (
	"context"
	"errors"
	"time"

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
func (d paypetalDeps) ensureCustomer(ctx context.Context, user *domain.User) (string, error) {
	if user.PayPetalCustomerID != "" {
		return user.PayPetalCustomerID, nil
	}
	if user.Phone == "" {
		return "", ErrPhoneRequired
	}
	customerID, err := d.client.CreateCustomer(ctx, user.Name, user.Email, user.Phone)
	if err != nil {
		return "", err
	}
	user.PayPetalCustomerID = customerID
	if err := d.userRepo.Update(ctx, user); err != nil {
		return "", err
	}
	return customerID, nil
}

// requirePayoutAccount gates any flow where user is about to become the
// counterparty on a new escrow agreement (the party who'd eventually be
// paid out to) — checked before creating the agreement, not after, since a
// missing payout account can't be fixed by anyone except that user.
func (d paypetalDeps) requirePayoutAccount(user *domain.User) error {
	if user.PayoutAccount == nil {
		return ErrPayoutAccountRequired
	}
	return nil
}

// escrowReferenceStale is the abandoned-checkout cutoff shared by every
// "revert if nobody paid" sweep (job hires, milestone funding).
const escrowReferenceStaleAfter = 30 * time.Minute
