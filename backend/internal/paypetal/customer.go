package paypetal

import (
	"context"
	"net/http"
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
