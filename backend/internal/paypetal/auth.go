package paypetal

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

type loginResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireAt    string `json:"expireAt"`
}

// authToken returns a cached token if it still has more than 5 minutes of
// life left, otherwise logs in again. Tokens are valid 7 days, so this is
// almost always a cache hit — logging in on every call would add an
// avoidable network round trip (and failure mode) to every single request.
func (c *Client) authToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-5*time.Minute)) {
		return c.token, nil
	}

	creds := base64.StdEncoding.EncodeToString([]byte(c.secretKey + ":" + c.appID))
	status, body, err := c.rawRequest(ctx, http.MethodPost, "/api/auth/login", map[string]string{"base64Hashed": creds}, "")
	if err != nil {
		return "", err
	}
	var login loginResponse
	if err := decode(body, status, &login); err != nil {
		return "", fmt.Errorf("paypetal login failed: %w", err)
	}

	expiry, err := time.Parse("2006-01-02T15:04:05.000Z07:00", login.ExpireAt)
	if err != nil {
		// If the expiry timestamp doesn't parse for any reason, fall back to
		// a conservative 1-hour cache rather than failing the whole login —
		// a shorter-than-necessary cache just means logging in a bit more
		// often, not a broken integration.
		expiry = time.Now().Add(1 * time.Hour)
	}

	c.token = login.AccessToken
	c.tokenExpiry = expiry
	return c.token, nil
}
