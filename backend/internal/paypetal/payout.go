package paypetal

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// Automatic payouts, the way PayPetal's guides describe them: GigPurse keeps
// a payout sub-wallet topped up, and each payout is a TrustCore agreement
// that sub-wallet funds on the spot, then releases to the payee's linked bank
// account. https://paypetal.readme.io/reference/createtrustcoreagreement

// SubWallet is one of GigPurse's sub-wallets. ID is what TrustCore takes as
// the initiator; AccountNumber is where to send money to top it up.
type SubWallet struct {
	ID            string  `json:"id"`
	RecordID      string  `json:"recordId"`
	Name          string  `json:"name"`
	Currency      string  `json:"currency"`
	Balance       float64 `json:"accountBalance"`
	BankName      string  `json:"bankName"`
	AccountNumber string  `json:"accountNumber"`
	AccountName   string  `json:"accountName"`
}

const subWalletPath = "/api/v1/account/subwallet"

// GetSubWallet looks a sub-wallet up by the reference it was created with,
// including its live balance. ErrNotFound if there is none.
func (c *Client) GetSubWallet(ctx context.Context, reference string) (*SubWallet, error) {
	var out SubWallet
	if err := c.do(ctx, http.MethodGet, subWalletPath+"?recordId="+url.QueryEscape(reference), nil, &out); err != nil {
		return nil, notFound(err, "subwallet_not_found")
	}
	return &out, nil
}

// CreateSubWallet needs PayPetal to have verified the business.
func (c *Client) CreateSubWallet(ctx context.Context, name, reference string) (*SubWallet, error) {
	body := map[string]string{"name": name, "currency": "NGN", "reference": reference}
	var out SubWallet
	if err := c.do(ctx, http.MethodPost, subWalletPath+"/create", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

const payoutAccountsPath = "/api/v1/escrow/payout-accounts/"

// LinkPayoutAccount sets the bank account TrustCore pays a customer to. The
// account must have been resolved with ValidateBankAccount in the last 10
// minutes. A customer has at most one; an existing one is replaced.
func (c *Client) LinkPayoutAccount(ctx context.Context, customerID, accountNumber, bankCode string) error {
	body := map[string]string{"accountNumber": accountNumber, "bankCode": bankCode}
	path := payoutAccountsPath + url.PathEscape(customerID)
	err := c.do(ctx, http.MethodPost, path, body, nil)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Code == "a_bank_account_has_already_been_added_for_this_customer" {
		return c.do(ctx, http.MethodPut, path, body, nil)
	}
	return err
}

// SubWalletPayout is a TrustCore agreement funded straight from a sub-wallet.
type SubWalletPayout struct {
	ReferenceID  string
	SubWalletID  string
	CustomerID   string // who gets paid
	Amount       string // FormatAmount
	Description  string
	RedirectURL  string // required by PayPetal even though nobody visits it
	PayoutRecord string // GigPurse's payout ID, kept in metadata
}

// CreateSubWalletPayout takes Amount plus PayPetal's fee out of the
// sub-wallet immediately. It returns nil if an agreement with this reference
// already exists, since a retry must never fund a second one; the caller
// checks with GetTrustCore that it's the agreement it expects.
// insufficient_subwallet_balance means nothing was taken.
func (c *Client) CreateSubWalletPayout(ctx context.Context, in SubWalletPayout) error {
	body := map[string]any{
		"referenceId":  in.ReferenceID,
		"initiator":    in.SubWalletID,
		"counterparty": in.CustomerID,
		"currency":     "NGN",
		"amount":       in.Amount,
		"description":  in.Description,
		"redirectUrl":  in.RedirectURL,
		"metadata":     map[string]string{"payoutId": in.PayoutRecord},
	}
	err := c.do(ctx, http.MethodPost, "/api/v1/escrow/trustcores", body, nil)
	if IsCode(err, "agreement_exists") {
		return nil
	}
	return err
}

// TrustCore is the state of a TrustCore agreement. PayoutStatus is NONE,
// PENDING, PROCESSING, COMPLETED, FAILED or REFUNDED (the bank sent it back);
// RefundStatus uses the same words for money returned to the sub-wallet.
type TrustCore struct {
	Reference    string
	Status       string // ONGOING or COMPLETED
	Amount       float64
	CustomerID   string // who it pays
	PayoutStatus string
	RefundStatus string
}

type trustCoreWire struct {
	Reference        string  `json:"reference"`
	Status           string  `json:"status"`
	Amount           float64 `json:"amount"`
	PayoutStatus     string  `json:"payoutStatus"`
	RefundStatus     string  `json:"refundStatus"`
	ReceiverCustomer *struct {
		CustomerID string `json:"customerId"`
	} `json:"receiverCustomer"`
}

func (w trustCoreWire) state() *TrustCore {
	t := &TrustCore{Reference: w.Reference, Status: w.Status, Amount: w.Amount, PayoutStatus: w.PayoutStatus, RefundStatus: w.RefundStatus}
	if w.ReceiverCustomer != nil {
		t.CustomerID = w.ReceiverCustomer.CustomerID
	}
	return t
}

const trustCorePath = "/api/v1/escrow/trustcore/"

// GetTrustCore is PayPetal's source of truth for a payout. ErrNotFound means
// no agreement exists, so no money has left the sub-wallet for it.
func (c *Client) GetTrustCore(ctx context.Context, reference string) (*TrustCore, error) {
	var out trustCoreWire
	if err := c.do(ctx, http.MethodGet, trustCorePath+url.PathEscape(reference), nil, &out); err != nil {
		return nil, notFound(err, "agreement_not_found")
	}
	return out.state(), nil
}

// CompleteTrustCore starts the bank transfer to the payee. It's
// asynchronous, and PayPetal refuses a second one while the first is in
// flight (payout_in_progress), so it can't pay twice.
func (c *Client) CompleteTrustCore(ctx context.Context, reference string) error {
	return c.do(ctx, http.MethodPut, trustCorePath+url.PathEscape(reference)+"/complete", nil, nil)
}

// RefundTrustCore returns an unpaid agreement's money to the sub-wallet that
// funded it. Asynchronous: wait for RefundStatus COMPLETED.
func (c *Client) RefundTrustCore(ctx context.Context, reference string) error {
	return c.do(ctx, http.MethodPut, trustCorePath+url.PathEscape(reference)+"/refund", nil, nil)
}

// ErrNotFound is PayPetal saying, definitively, that a thing doesn't exist.
var ErrNotFound = errors.New("paypetal: not found")

func notFound(err error, code string) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) && (apiErr.Code == code || (apiErr.Code == "" && apiErr.HTTPStatus == http.StatusNotFound)) {
		return ErrNotFound
	}
	return err
}

// IsCode reports whether err is PayPetal refusing with this error code.
func IsCode(err error, code string) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == code
}
