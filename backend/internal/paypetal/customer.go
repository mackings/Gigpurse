package paypetal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type customerResponse struct {
	CustomerID string `json:"customerId"`
}

// CreateCustomer registers a GigPurse user as a PayPetal customer. Called
// lazily the first time a user is party to any payment (never at signup) —
// see ensureCustomer in usecase/paypetal_helpers.go.
func (c *Client) CreateCustomer(ctx context.Context, fullname, email, phone string) (string, error) {
	body := map[string]string{"fullname": fullname}
	if email != "" {
		body["email"] = email
	}
	if phone != "" {
		body["phoneNumber"] = phone
	}
	var out customerResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/customer", body, &out); err != nil {
		return "", err
	}
	return out.CustomerID, nil
}

// UpdateCustomer patches an existing customer record — only fields passed
// non-empty are changed, per PayPetal's own "omitted fields retain their
// current values" behavior. Used to backfill a phone number onto a customer
// that was created earlier without one (e.g. a client whose PayPetal
// identity was created at hire time, before they ever added a phone via the
// payout-account flow).
func (c *Client) UpdateCustomer(ctx context.Context, customerID, fullname, email, phone string) error {
	body := map[string]string{"fullname": fullname}
	if email != "" {
		body["email"] = email
	}
	if phone != "" {
		body["phoneNumber"] = phone
	}
	return c.do(ctx, http.MethodPut, "/api/v1/customer/"+customerID, body, nil)
}

// Customer is one entry from ListCustomers.
type Customer struct {
	CustomerID  string `json:"customerId"`
	Fullname    string `json:"fullname"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

// CustomerExists reports whether this merchant account still knows
// customerID. False only on PayPetal's definite customer_not_found — a
// network failure or any other error is returned as an error, so a caller
// never mistakes an outage for a missing customer.
func (c *Client) CustomerExists(ctx context.Context, customerID string) (bool, error) {
	err := c.do(ctx, http.MethodGet, "/api/v1/customer/"+url.PathEscape(customerID), nil, nil)
	if err == nil {
		return true, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && (apiErr.Code == "customer_not_found" || strings.EqualFold(apiErr.Message, "customer not found")) {
		return false, nil
	}
	return false, err
}

type customerPage struct {
	Content    []Customer `json:"content"`
	TotalPages int        `json:"totalPages"`
}

// ListCustomers returns every customer under the merchant account. PayPetal
// has no lookup-by-email/phone endpoint, so this is the fallback
// ensureCustomer uses when CreateCustomer reports "already exists" — it
// happens whenever GigPurse's own record of a user's customerId was lost
// (e.g. a local DB reset) while the same email/phone's PayPetal customer
// still exists from before. The endpoint is paged (1-based); this walks every
// page.
func (c *Client) ListCustomers(ctx context.Context) ([]Customer, error) {
	var all []Customer
	for page := 1; ; page++ {
		var out customerPage
		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/customer/all?page=%d&size=100", page), nil, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Content...)
		if len(out.Content) == 0 || page >= out.TotalPages {
			return all, nil
		}
	}
}

// DeleteCustomer permanently removes a customer record — irreversible, per
// PayPetal's own docs. Existing escrow agreements referencing this
// customer aren't affected, but the customer profile itself becomes
// unlookupable afterward. Not used by any GigPurse feature today — this
// exists purely for ops cleanup (e.g. clearing sandbox test data), called
// from a one-off script rather than through the paypetal.API interface.
func (c *Client) DeleteCustomer(ctx context.Context, customerID string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/customer/"+customerID, nil, nil)
}
