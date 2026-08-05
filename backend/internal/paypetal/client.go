// Package paypetal is a thin client for PayPetal's Merchant API — the
// TrustCore escrow product specifically (see https://paypetal.readme.io).
// It is the only place in the codebase that speaks PayPetal's wire format
// (its two different response envelope shapes, its kobo amounts, its JWT
// login exchange); everything above this package deals in plain Naira
// floats and GigPurse's own domain types.
package paypetal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// API is the subset of Client that usecases depend on — extracted purely so
// tests can substitute an in-memory fake (see repository/memory/paypetal_fake.go)
// without spinning up real HTTP calls.
type API interface {
	CreateCustomer(ctx context.Context, fullname, email, phone string) (customerID string, err error)
	UpdateCustomer(ctx context.Context, customerID, fullname, email, phone string) error

	ListBanks(ctx context.Context) ([]Bank, error)
	ValidateBankAccount(ctx context.Context, accountNumber, bankCode string) (accountName string, err error)
	LinkPayoutAccount(ctx context.Context, customerID, accountNumber, bankCode string) error

	CreateTrustCoreAgreement(ctx context.Context, in CreateAgreementInput) (*AgreementResult, error)
	GetTrustCoreAgreement(ctx context.Context, reference string) (*AgreementState, error)
	CompleteTrustCoreAgreement(ctx context.Context, reference string) error
	RefundTrustCoreAgreement(ctx context.Context, reference string) error
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	secretKey  string
	appID      string

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

var _ API = (*Client)(nil)

func NewClient(baseURL, secretKey, appID string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
		secretKey:  secretKey,
		appID:      appID,
	}
}

// do performs an authenticated call and decodes the response into out
// (which may be nil for calls with no meaningful success payload, like
// complete/refund). A 401 clears the cached token and retries exactly once
// with a fresh login, to absorb early token invalidation/clock skew without
// risking an infinite retry loop on a genuinely bad request.
func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	token, err := c.authToken(ctx)
	if err != nil {
		return err
	}
	status, respBody, err := c.rawRequest(ctx, method, path, body, token)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized {
		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()
		token, err = c.authToken(ctx)
		if err != nil {
			return err
		}
		status, respBody, err = c.rawRequest(ctx, method, path, body, token)
		if err != nil {
			return err
		}
	}
	return decode(respBody, status, out)
}

func (c *Client) rawRequest(ctx context.Context, method, path string, body interface{}, token string) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("paypetal: encoding request: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, nil, fmt.Errorf("paypetal: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("paypetal: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("paypetal: reading response: %w", err)
	}
	return resp.StatusCode, respBody, nil
}
