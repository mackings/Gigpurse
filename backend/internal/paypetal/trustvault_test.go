package paypetal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// Request bodies, paths and response shapes as documented by PayPetal.
func TestTrustVaultWireFormat(t *testing.T) {
	var method, path string
	var body map[string]string
	reply := ""
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path, body = r.Method, r.URL.RequestURI(), nil
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		fmt.Fprint(w, reply)
	})
	ctx := context.Background()
	expect := func(m, p string) {
		t.Helper()
		if method != m || path != p {
			t.Fatalf("got %s %s, want %s %s", method, path, m, p)
		}
	}

	reply = `{"status":"success","responseCode":"00","message":"Agreement created successfully","responseData":{"paymentInfo":{"currency":"NGN","totalAmount":152000,"amount":150000,"fee":2000,"paymentUrl":"https://x/pay?pid=ms1"},"agreement":{"idempotencyKey":"ms1","reference":"2611161035129048C1E7Q2M0D","status":"CREATED"}}}`
	res, err := c.CreateVaultAgreement(ctx, CreateVaultInput{ReferenceID: "ms1", CustomerID: "cust_1", Currency: "NGN", Amount: "525.00", Description: "d", RedirectURL: "https://r"})
	if err != nil {
		t.Fatal(err)
	}
	expect("POST", "/api/v1/escrow/trustvault/agreement")
	want := map[string]string{"referenceId": "ms1", "customerId": "cust_1", "currency": "NGN", "amount": "525.00", "type": "ONETIME", "feePayer": "SENDER", "description": "d", "redirectUrl": "https://r"}
	if fmt.Sprint(body) != fmt.Sprint(want) {
		t.Fatalf("body %v", body)
	}
	if res.PaymentURL != "https://x/pay?pid=ms1" || res.Reference != "2611161035129048C1E7Q2M0D" || res.Status != "CREATED" || res.TotalAmount != 152000 {
		t.Fatalf("result %#v", res)
	}

	reply = `{"status":"success","responseData":{"idempotencyKey":"ms1","status":"PAID","releasedAmount":0}}`
	state, err := c.GetVaultAgreement(ctx, "ms1")
	if err != nil || state.Status != "PAID" {
		t.Fatalf("%#v %v", state, err)
	}
	expect("GET", "/api/v1/escrow/trustvault/agreement/ms1")

	reply = `{"status":"success","responseData":{"message":"Authorization code sent to the customer email","releaseAmount":150000}}`
	if err := c.SendVaultReleaseCode(ctx, "ms1"); err != nil {
		t.Fatal(err)
	}
	expect("PUT", "/api/v1/escrow/trustvault/agreement/ms1/release/auth")

	reply = `{"status":"success","responseData":{"status":"COMPLETED"}}`
	if err := c.ReleaseVaultAgreement(ctx, "ms1", "483921"); err != nil {
		t.Fatal(err)
	}
	expect("PUT", "/api/v1/escrow/trustvault/agreement/ms1/release")
	if fmt.Sprint(body) != fmt.Sprint(map[string]string{"code": "483921"}) {
		t.Fatalf("body %v", body)
	}
}
