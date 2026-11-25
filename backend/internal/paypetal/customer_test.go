package paypetal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"success","responseData":{"accessToken":"t","expireAt":"2099-01-01T00:00:00Z"}}`)
	})
	mux.HandleFunc("/", handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(server.URL, "key", "app")
}

// The live endpoint answers with a page object, not a bare array.
func TestListCustomers_WalksEveryPage(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		fmt.Fprintf(w, `{"status":"success","responseData":{"content":[{"customerId":"cust_%s","email":"%s@example.com"}],"totalPages":3,"pageNumber":%s}}`, page, page, page)
	})
	got, err := c.ListCustomers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].CustomerID != "cust_1" || got[2].CustomerID != "cust_3" {
		t.Fatalf("expected three pages of customers, got %#v", got)
	}
}

func TestListCustomers_Empty(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"success","responseData":{"content":[],"totalPages":0}}`)
	})
	if got, err := c.ListCustomers(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestCustomerExists(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		exists, ok bool
	}{
		{"found", `{"status":"success","responseData":{"customerId":"cust_1"}}`, 200, true, true},
		{"not found", `{"status":"error","message":"Customer not found","responseData":{"code":"customer_not_found","message":"Customer not found"}}`, 404, false, true},
		{"outage is not a missing customer", `<html>bad gateway</html>`, 502, false, false},
		{"rate limited is not a missing customer", `{"status":"error","message":"Too many requests","responseData":{"code":"rate_limited","message":"Too many requests"}}`, 429, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			})
			exists, err := c.CustomerExists(context.Background(), "cust_1")
			if exists != tc.exists || (err == nil) != tc.ok {
				t.Fatalf("got exists=%v err=%v", exists, err)
			}
		})
	}
}
