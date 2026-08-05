package paypetal

import (
	"context"
	"net/http"
	"net/url"
)

// CreateAgreementInput describes a new TrustCore escrow agreement.
// AmountKobo must already be in the smallest currency unit — build it with
// NairaToKobo right before calling, so the conversion is visible at the
// call site rather than hidden inside this package.
type CreateAgreementInput struct {
	ReferenceID            string
	InitiatorCustomerID    string
	CounterpartyCustomerID string
	Currency               string // "NGN" or "USD"
	AmountKobo             string
	Description            string
	RedirectURL            string
}

type AgreementResult struct {
	PaymentURL    string
	TransactionID string
	Reference     string
	Status        string
}

type createAgreementResponse struct {
	PaymentURL string `json:"paymentUrl"`
	Data       struct {
		TransactionID string `json:"transactionId"`
		Reference     string `json:"reference"`
		Status        string `json:"status"`
	} `json:"data"`
}

// CreateTrustCoreAgreement establishes a new merchant-controlled escrow
// agreement and returns a hosted payment URL for the initiator to fund it.
func (c *Client) CreateTrustCoreAgreement(ctx context.Context, in CreateAgreementInput) (*AgreementResult, error) {
	body := map[string]string{
		"referenceId":  in.ReferenceID,
		"initiator":    in.InitiatorCustomerID,
		"counterparty": in.CounterpartyCustomerID,
		"currency":     in.Currency,
		"amount":       in.AmountKobo,
		"description":  in.Description,
		"redirectUrl":  in.RedirectURL,
	}
	var out createAgreementResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/escrow/trustcores", body, &out); err != nil {
		return nil, err
	}
	return &AgreementResult{
		PaymentURL:    out.PaymentURL,
		TransactionID: out.Data.TransactionID,
		Reference:     out.Data.Reference,
		Status:        out.Data.Status,
	}, nil
}

// AgreementState is a TrustCore agreement's current status as reported by
// PayPetal — this is the one authoritative source of truth GigPurse acts
// on; it never trusts a webhook payload's claimed status directly (see the
// webhook handler for why).
type AgreementState struct {
	Reference     string
	TransactionID string
	Status        string // e.g. "ONGOING" once funded
	PayoutStatus  string // "NONE" | "PENDING" | ...
	RefundStatus  string // "NONE" | "PENDING" | ...
}

type getAgreementResponse struct {
	Reference     string `json:"reference"`
	TransactionID string `json:"transactionId"`
	Status        string `json:"status"`
	PayoutStatus  string `json:"payoutStatus"`
	RefundStatus  string `json:"refundStatus"`
}

// GetTrustCoreAgreement fetches an agreement's current state by its
// referenceId or PayPetal transactionId.
func (c *Client) GetTrustCoreAgreement(ctx context.Context, reference string) (*AgreementState, error) {
	path := "/api/v1/escrow/trustcore?reference=" + url.QueryEscape(reference)
	var out getAgreementResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &AgreementState{
		Reference:     out.Reference,
		TransactionID: out.TransactionID,
		Status:        out.Status,
		PayoutStatus:  out.PayoutStatus,
		RefundStatus:  out.RefundStatus,
	}, nil
}

// CompleteTrustCoreAgreement releases the escrowed amount to the
// counterparty. Requires the agreement to be ONGOING, the counterparty to
// have a payout account on file, and payoutStatus to still be NONE — a
// failure here typically surfaces as an *APIError with Code
// "missing_payout_account", "payout_in_progress", etc.
func (c *Client) CompleteTrustCoreAgreement(ctx context.Context, reference string) error {
	return c.do(ctx, http.MethodPut, "/api/v1/escrow/trustcore/"+url.PathEscape(reference)+"/complete", nil, nil)
}

// RefundTrustCoreAgreement returns the full escrowed amount to the
// initiator. Full refund only — PayPetal doesn't support partial refunds.
func (c *Client) RefundTrustCoreAgreement(ctx context.Context, reference string) error {
	return c.do(ctx, http.MethodPut, "/api/v1/escrow/trustcore/"+url.PathEscape(reference)+"/refund", nil, nil)
}
