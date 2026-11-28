package paypetal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// Request bodies, paths and responses as PayPetal's docs give them for
// sub-wallets, payout accounts and sub-wallet-funded TrustCore.
func TestPayoutWireFormat(t *testing.T) {
	type call struct {
		method, path string
		body         map[string]any
	}
	var calls []call
	var replies []string
	status := []int{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		calls = append(calls, call{r.Method, r.URL.RequestURI(), body})
		code, reply := http.StatusOK, replies[0]
		if len(status) > 0 {
			code, status = status[0], status[1:]
		}
		replies = replies[1:]
		w.WriteHeader(code)
		fmt.Fprint(w, reply)
	})
	ctx := context.Background()
	last := func(m, p string) call {
		t.Helper()
		got := calls[len(calls)-1]
		if got.method != m || got.path != p {
			t.Fatalf("got %s %s, want %s %s", got.method, got.path, m, p)
		}
		return got
	}

	// Sub-wallet lookup and creation.
	replies = []string{`{"status":"success","responseCode":"00","message":"Records retrieved","responseData":{"id":"b7e2c9a4-1f63-4d8e-9c05-2a7f8e3d6b11","accountBalance":15000,"currency":"NGN","recordId":"gigpurse-payouts","name":"GigPurse Payouts","bankName":"9Payment (9PSB)","accountNumber":"5031681901","accountName":"GIGPURSE-GIGPURSE PAYOUTS"}}`}
	sw, err := c.GetSubWallet(ctx, "gigpurse-payouts")
	if err != nil || sw.ID != "b7e2c9a4-1f63-4d8e-9c05-2a7f8e3d6b11" || sw.Balance != 15000 || sw.AccountNumber != "5031681901" {
		t.Fatalf("%#v %v", sw, err)
	}
	last("GET", "/api/v1/account/subwallet?recordId=gigpurse-payouts")

	replies, status = []string{`{"status":"error","responseCode":"404","message":"Subwallet not found","responseData":{"type":"resource_not_found","code":"subwallet_not_found","message":"Subwallet not found","param":null}}`}, []int{404}
	if _, err := c.GetSubWallet(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	replies = []string{`{"status":"success","responseCode":"00","message":"Subwallet created","responseData":{"id":"sw1","accountBalance":0,"currency":"NGN","recordId":"gigpurse-payouts","name":"GigPurse Payouts","bankName":"9Payment (9PSB)","accountNumber":"5031681901","accountName":"X"}}`}
	if sw, err := c.CreateSubWallet(ctx, "GigPurse Payouts", "gigpurse-payouts"); err != nil || sw.ID != "sw1" {
		t.Fatalf("%#v %v", sw, err)
	}
	if b := last("POST", "/api/v1/account/subwallet/create").body; fmt.Sprint(b) != fmt.Sprint(map[string]any{"name": "GigPurse Payouts", "currency": "NGN", "reference": "gigpurse-payouts"}) {
		t.Fatalf("body %v", b)
	}

	// Linking a payout account: add, or replace one that's already there.
	linked := `{"status":"success","responseCode":"00","message":"Operation successful","responseData":{"accountName":"Tunde Bakare","accountNumber":"0123456789","bankName":"Guaranty Trust Bank","bankCode":"000013"}}`
	replies = []string{linked}
	if err := c.LinkPayoutAccount(ctx, "cust_1", "0123456789", "000013"); err != nil {
		t.Fatal(err)
	}
	if b := last("POST", "/api/v1/escrow/payout-accounts/cust_1").body; fmt.Sprint(b) != fmt.Sprint(map[string]any{"accountNumber": "0123456789", "bankCode": "000013"}) {
		t.Fatalf("body %v", b)
	}
	replies, status = []string{`{"status":"error","responseCode":"400","message":"A bank account has already been added for this customer","responseData":{"type":"conflict_error","code":"a_bank_account_has_already_been_added_for_this_customer","message":"A bank account has already been added for this customer","param":null}}`, linked}, []int{400, 200}
	if err := c.LinkPayoutAccount(ctx, "cust_1", "2045817736", "000014"); err != nil {
		t.Fatal(err)
	}
	last("PUT", "/api/v1/escrow/payout-accounts/cust_1")

	// Funding from the sub-wallet.
	replies = []string{`{"status":"success","responseCode":"00","message":"Records retrieved","responseData":{"paymentUrl":null,"data":{"currency":"NGN","amount":450,"fee":10,"totalAmount":460,"transactionId":"T1","reference":"ptms1","status":"SUCCESS","paymentMethod":"SUBWALLET","operationType":"TRUSTCORE"}}}`}
	in := SubWalletPayout{ReferenceID: "ptms1", SubWalletID: "sw1", CustomerID: "cust_1", Amount: "450.00", Description: "GigPurse payout: Set", RedirectURL: "https://g/wallet", PayoutRecord: "ms1:talent_payout"}
	if err := c.CreateSubWalletPayout(ctx, in); err != nil {
		t.Fatal(err)
	}
	b := last("POST", "/api/v1/escrow/trustcores").body
	want := map[string]any{"referenceId": "ptms1", "initiator": "sw1", "counterparty": "cust_1", "currency": "NGN", "amount": "450.00", "description": "GigPurse payout: Set", "redirectUrl": "https://g/wallet", "metadata": map[string]any{"payoutId": "ms1:talent_payout"}}
	if fmt.Sprint(b) != fmt.Sprint(want) {
		t.Fatalf("body %v", b)
	}
	_, hasCharge := b["merchantCharge"]
	if hasCharge {
		t.Fatal("no merchant charge on payouts")
	}
	replies, status = []string{`{"status":"error","responseCode":"400","message":"A TrustCore agreement with reference 'ptms1' already exists.","responseData":{"type":"conflict_error","code":"agreement_exists","message":"A TrustCore agreement with reference 'ptms1' already exists.","param":null}}`}, []int{400}
	if err := c.CreateSubWalletPayout(ctx, in); err != nil {
		t.Fatalf("an existing agreement is not an error: %v", err)
	}
	replies, status = []string{`{"status":"error","responseCode":"400","message":"The subwallet's balance is too low to fund this escrow.","responseData":{"type":"payment_error","code":"insufficient_subwallet_balance","message":"The subwallet's balance is too low to fund this escrow.","param":null}}`}, []int{400}
	if err := c.CreateSubWalletPayout(ctx, in); !IsCode(err, "insufficient_subwallet_balance") {
		t.Fatalf("want insufficient_subwallet_balance, got %v", err)
	}

	// State, transfer, refund.
	replies = []string{`{"status":"success","responseCode":"00","message":"Records retrieved","responseData":{"reference":"ptms1","transactionId":"T1","currency":"NGN","amount":450,"merchantChargeAmount":0,"description":"d","payerCustomer":null,"receiverCustomer":{"fullname":"Tunde","customerId":"cust_1","email":"t@x","phoneNumber":"+234","createdAt":"2026-11-16T09:03:27.115+00:00"},"status":"ONGOING","feePayer":"MERCHANT","payoutStatus":"NONE","refundStatus":"NONE","createdAt":"2026-11-16T10:02:33.147+00:00"}}`}
	tc, err := c.GetTrustCore(ctx, "ptms1")
	if err != nil || tc.Amount != 450 || tc.CustomerID != "cust_1" || tc.PayoutStatus != "NONE" || tc.RefundStatus != "NONE" || tc.Status != "ONGOING" {
		t.Fatalf("%#v %v", tc, err)
	}
	last("GET", "/api/v1/escrow/trustcore/ptms1")
	replies, status = []string{`{"status":"error","responseCode":"404","message":"No agreement found with reference or transaction id 'ptms2'.","responseData":{"type":"resource_not_found","code":"agreement_not_found","message":"No agreement found","param":null}}`}, []int{404}
	if _, err := c.GetTrustCore(ctx, "ptms2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	replies, status = []string{`{"status":"error","responseCode":"500","message":"Internal error"}`}, []int{500}
	if _, err := c.GetTrustCore(ctx, "ptms1"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("an outage must not read as not found, got %v", err)
	}

	replies = []string{`{"status":"success","responseCode":"00","message":"Payout initiated successfully","responseData":{"reference":"ptms1","payoutStatus":"PENDING","payoutAmount":450}}`}
	if err := c.CompleteTrustCore(ctx, "ptms1"); err != nil {
		t.Fatal(err)
	}
	last("PUT", "/api/v1/escrow/trustcore/ptms1/complete")
	replies = []string{`{"status":"success","responseCode":"00","message":"Refund initiated successfully","responseData":{"reference":"ptms1","refundStatus":"PENDING"}}`}
	if err := c.RefundTrustCore(ctx, "ptms1"); err != nil {
		t.Fatal(err)
	}
	last("PUT", "/api/v1/escrow/trustcore/ptms1/refund")
}
