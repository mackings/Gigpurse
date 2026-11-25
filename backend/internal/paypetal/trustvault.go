package paypetal

import (
	"context"
	"net/http"
	"net/url"
)

// CreateVaultInput describes a new one-time TrustVault agreement. There's no
// receiving party: the customer pays Amount (a Naira string built with
// FormatAmount), and on release all of it lands in GigPurse's own wallet.
type CreateVaultInput struct {
	ReferenceID string
	CustomerID  string // who pays, and who PayPetal emails release codes to
	Currency    string
	Amount      string
	Description string
	RedirectURL string
}

type VaultResult struct {
	PaymentURL  string
	Reference   string // PayPetal's own reference, not ours
	Status      string // "CREATED" until paid
	TotalAmount float64
}

// VaultState is a TrustVault agreement's current status. Status is CREATED
// (unpaid), PAID (held), or COMPLETED (everything released to our wallet).
type VaultState struct {
	Status         string
	ReleasedAmount float64
}

// Paid reports whether the client's money has reached escrow (including if
// it has since been released).
func (s *VaultState) Paid() bool { return s.Status == "PAID" || s.Status == "COMPLETED" }

const vaultPath = "/api/v1/escrow/trustvault/agreement"

// CreateVaultAgreement opens a customer-controlled escrow and returns the
// hosted payment URL. Idempotent on ReferenceID — resending one returns the
// original agreement.
func (c *Client) CreateVaultAgreement(ctx context.Context, in CreateVaultInput) (*VaultResult, error) {
	body := map[string]string{
		"referenceId": in.ReferenceID,
		"customerId":  in.CustomerID,
		"currency":    in.Currency,
		"amount":      in.Amount,
		"type":        "ONETIME",
		"feePayer":    "SENDER",
		"description": in.Description,
		"redirectUrl": in.RedirectURL,
	}
	var out struct {
		PaymentInfo struct {
			PaymentURL  string  `json:"paymentUrl"`
			TotalAmount float64 `json:"totalAmount"`
		} `json:"paymentInfo"`
		Agreement struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
		} `json:"agreement"`
	}
	if err := c.do(ctx, http.MethodPost, vaultPath, body, &out); err != nil {
		return nil, err
	}
	return &VaultResult{
		PaymentURL:  out.PaymentInfo.PaymentURL,
		Reference:   out.Agreement.Reference,
		Status:      out.Agreement.Status,
		TotalAmount: out.PaymentInfo.TotalAmount,
	}, nil
}

func (c *Client) GetVaultAgreement(ctx context.Context, reference string) (*VaultState, error) {
	var out struct {
		Status         string  `json:"status"`
		ReleasedAmount float64 `json:"releasedAmount"`
	}
	if err := c.do(ctx, http.MethodGet, vaultPath+"/"+url.PathEscape(reference), nil, &out); err != nil {
		return nil, err
	}
	return &VaultState{Status: out.Status, ReleasedAmount: out.ReleasedAmount}, nil
}

// SendVaultReleaseCode emails the paying customer a 6-digit code, valid 20
// minutes, that authorises releasing everything still held.
func (c *Client) SendVaultReleaseCode(ctx context.Context, reference string) error {
	return c.do(ctx, http.MethodPut, vaultPath+"/"+url.PathEscape(reference)+"/release/auth", nil, nil)
}

// ReleaseVaultAgreement moves everything held to GigPurse's wallet, using the
// code the customer received. Nothing is sent to the talent by this call.
func (c *Client) ReleaseVaultAgreement(ctx context.Context, reference, code string) error {
	return c.do(ctx, http.MethodPut, vaultPath+"/"+url.PathEscape(reference)+"/release", map[string]string{"code": code}, nil)
}
