package paypetal

import (
	"context"
	"net/http"
)

type Bank struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// ListBanks returns every Nigerian bank PayPetal supports, for a bank-picker
// dropdown ahead of ValidateBankAccount/LinkPayoutAccount.
func (c *Client) ListBanks(ctx context.Context) ([]Bank, error) {
	var out []Bank
	if err := c.do(ctx, http.MethodGet, "/api/v1/account/banks", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type bankValidateResponse struct {
	AccountName string `json:"accountName"`
}

// ValidateBankAccount resolves an account number to its registered holder
// name. The caller must show this name back to the user for confirmation
// before saving it — PayPetal cannot reverse a payout sent to the wrong
// account.
func (c *Client) ValidateBankAccount(ctx context.Context, accountNumber, bankCode string) (string, error) {
	body := map[string]string{"accountNumber": accountNumber, "bankCode": bankCode}
	var out bankValidateResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/account/bank/validate", body, &out); err != nil {
		return "", err
	}
	return out.AccountName, nil
}

// LinkPayoutAccount associates a pre-validated bank account with a customer
// so they can receive TrustCore releases/refunds. Call ValidateBankAccount
// first — this endpoint expects an already-validated account.
func (c *Client) LinkPayoutAccount(ctx context.Context, customerID, accountNumber, bankCode string) error {
	body := map[string]string{"accountNumber": accountNumber, "bankCode": bankCode}
	return c.do(ctx, http.MethodPost, "/api/v1/escrow/trustcore/payout/"+customerID, body, nil)
}

// RemovePayoutAccount unlinks a customer's payout account — discovered as a
// precondition for DeleteCustomer: PayPetal refuses to delete a customer
// that still has one linked (returned as a misleading 401 rather than 409,
// confirmed live against sandbox). Also used by payoutAccountUsecase.Link to
// self-heal a customer PayPetal already has a (possibly stale) bank account
// on file for — see isBankAlreadyLinkedError.
func (c *Client) RemovePayoutAccount(ctx context.Context, customerID string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/escrow/trustcore/payout/"+customerID, nil, nil)
}
