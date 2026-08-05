package paypetal

import "fmt"

// APIError wraps a failed PayPetal call. Code is PayPetal's own error code
// where available (e.g. "missing_payout_account", "agreement_not_found") so
// callers can branch on specific failure reasons instead of string-matching
// Message, which is free text meant for logs/humans, not control flow.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("paypetal: %s (%s, HTTP %d)", e.Message, e.Code, e.HTTPStatus)
	}
	return fmt.Sprintf("paypetal: %s (HTTP %d)", e.Message, e.HTTPStatus)
}
