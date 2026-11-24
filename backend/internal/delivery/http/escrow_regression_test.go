package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

// escrowFixture is a client and a talent with payout accounts and one active
// direct-hire contract between them.
type escrowFixture struct {
	*apiClient
	app                            *testApp
	clientID, musicianID           string
	clientTok, musicianTok, admTok string
	contractID                     string
}

func newEscrowFixture(t *testing.T) *escrowFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", "escrow-regression-secret")
	t.Setenv("ALLOW_ADMIN_SIGNUP", "true")
	app := newTestApp()
	server := httptest.NewServer(app.mux)
	t.Cleanup(server.Close)
	c := &apiClient{t: t, baseURL: server.URL, http: server.Client()}
	f := &escrowFixture{apiClient: c, app: app}

	join := func(email, role, code string) (string, string) {
		u := c.signup(email, "password123", role, "Demo "+role)
		c.verifyEmail(app, u.ID, email, code)
		return u.ID, c.login(email, "password123")
	}
	f.clientID, f.clientTok = join("client@example.com", "client", "111111")
	f.musicianID, f.musicianTok = join("musician@example.com", "musician", "222222")
	_, f.admTok = join("admin@example.com", "admin", "333333")

	for _, tok := range []string{f.clientTok, f.musicianTok} {
		c.post("/payout-account", tok, map[string]any{
			"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0123456789",
		}, http.StatusOK, nil)
	}

	var hire domain.DirectHireRequest
	c.post("/direct-hires", f.clientTok, map[string]any{
		"musician_id": f.musicianID, "title": "Wedding set", "description": "Two hours", "price": 500,
	}, http.StatusCreated, &hire)
	c.post("/direct-hires/respond", f.musicianTok, map[string]any{"request_id": hire.ID, "decision": "accepted"}, http.StatusOK, &hire)
	f.contractID = hire.ContractID
	return f
}

func (f *escrowFixture) acceptedMilestone(amount float64) string {
	var ms []domain.Milestone
	f.post("/milestones", f.clientTok, map[string]any{
		"contract_id": f.contractID, "milestones": []map[string]any{{"title": "Set", "amount": amount}},
	}, http.StatusCreated, &ms)
	f.post("/milestones/accept", f.musicianTok, map[string]any{"contract_id": f.contractID, "milestone_id": ms[0].ID}, http.StatusOK, nil)
	return ms[0].ID
}

func (f *escrowFixture) startFunding(milestoneID string) string {
	var start struct {
		Reference string `json:"reference"`
	}
	f.post("/milestones/fund", f.clientTok, map[string]any{"contract_id": f.contractID, "milestone_id": milestoneID}, http.StatusOK, &start)
	return start.Reference
}

func (f *escrowFixture) fundedMilestone(amount float64) (id, reference string) {
	id = f.acceptedMilestone(amount)
	reference = f.startFunding(id)
	f.app.paypetal.SimulatePaymentCompleted(reference)
	f.post("/milestones/fund/finalize", f.clientTok, map[string]any{"reference": reference}, http.StatusOK, nil)
	return id, reference
}

func (f *escrowFixture) milestone(id string) domain.Milestone {
	f.t.Helper()
	var ms []domain.Milestone
	f.get("/milestones?contract_id="+f.contractID, f.clientTok, http.StatusOK, &ms)
	for _, m := range ms {
		if m.ID == id {
			return m
		}
	}
	f.t.Fatalf("milestone %s not found", id)
	return domain.Milestone{}
}

// The /musicians endpoints need no login, so they must never carry contact or
// bank details.
func TestMusicians_PublicEndpointsHidePrivateDetails(t *testing.T) {
	f := newEscrowFixture(t)
	f.put("/users/profile", f.musicianTok, map[string]any{
		"name": "Demo musician", "bio": "Plays guitar", "location": "Lagos",
		"musician_profile": map[string]any{"genres": []string{"Afrobeats"}, "saved_job_ids": []string{"job_secret"}},
	}, http.StatusOK, nil)

	for _, path := range []string{"/musicians", "/musicians/" + f.musicianID} {
		var raw json.RawMessage
		f.get(path, "", http.StatusOK, &raw)
		body := string(raw)
		for _, secret := range []string{"email", "phone", "payout_account", "account_number", "0123456789", "Fake Test Bank", "terms_accepted_at", "job_secret"} {
			if strings.Contains(body, secret) {
				t.Errorf("%s exposes %q: %s", path, secret, body)
			}
		}
		if !strings.Contains(body, "Plays guitar") || !strings.Contains(body, "Afrobeats") {
			t.Errorf("%s lost its public profile fields: %s", path, body)
		}
	}

	// The owner still sees their own.
	var me domain.User
	f.get("/users/profile", f.musicianTok, http.StatusOK, &me)
	if me.Email == "" || me.PayoutAccount == nil {
		t.Fatalf("own profile should still include private details, got %#v", me)
	}
}

func TestOpenDispute_LocksFundedMilestones(t *testing.T) {
	for _, tc := range []struct {
		name   string
		award  float64
		status string
	}{
		{"full release", 450, "released"},
		{"full refund", 0, "refunded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newEscrowFixture(t)
			id, ref := f.fundedMilestone(500)
			body := map[string]any{"contract_id": f.contractID, "milestone_id": id}

			var dispute domain.Dispute
			f.post("/disputes", f.musicianTok, map[string]any{"contract_id": f.contractID, "reason": "Not paid"}, http.StatusCreated, &dispute)

			if m := f.milestone(id); m.Status != "disputed" || m.DisputeID != dispute.ID {
				t.Fatalf("expected the funded milestone locked under %s, got %q/%q", dispute.ID, m.Status, m.DisputeID)
			}
			f.post("/milestones/release", f.clientTok, body, http.StatusBadRequest, nil)
			f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref}, http.StatusBadRequest, nil)
			if m := f.milestone(id); m.Status != "disputed" {
				t.Fatalf("milestone escaped the dispute: %q", m.Status)
			}

			var holdings []domain.EscrowHolding
			f.get("/wallet/escrow", f.clientTok, http.StatusOK, &holdings)
			if len(holdings) != 1 || !holdings[0].Disputed {
				t.Fatalf("expected the holding flagged as disputed, got %#v", holdings)
			}

			f.post("/disputes/join", f.admTok, map[string]any{"dispute_id": dispute.ID}, http.StatusOK, nil)
			f.post("/admin/disputes/resolve", f.admTok, map[string]any{
				"dispute_id": dispute.ID, "resolution": "Ruled", "talent_amount": tc.award,
			}, http.StatusOK, nil)
			if m := f.milestone(id); m.Status != tc.status {
				t.Fatalf("expected %q after ruling, got %q", tc.status, m.Status)
			}
		})
	}
}

// A contract-wide dispute must not settle money a separate milestone dispute
// already locked.
func TestResolveDispute_LeavesOtherDisputesMoneyAlone(t *testing.T) {
	f := newEscrowFixture(t)
	first, _ := f.fundedMilestone(500)
	second, _ := f.fundedMilestone(300)
	f.post("/milestones/cancel", f.musicianTok, map[string]any{"contract_id": f.contractID, "milestone_id": first}, http.StatusOK, nil)

	var wide domain.Dispute
	f.post("/disputes", f.clientTok, map[string]any{"contract_id": f.contractID, "reason": "Late"}, http.StatusCreated, &wide)
	f.post("/disputes/join", f.admTok, map[string]any{"dispute_id": wide.ID}, http.StatusOK, nil)
	f.post("/admin/disputes/resolve", f.admTok, map[string]any{"dispute_id": wide.ID, "resolution": "Refund", "talent_amount": 0}, http.StatusOK, nil)

	if m := f.milestone(second); m.Status != "refunded" {
		t.Fatalf("expected the second milestone refunded, got %q", m.Status)
	}
	if m := f.milestone(first); m.Status != "disputed" {
		t.Fatalf("expected the first milestone still under its own dispute, got %q", m.Status)
	}
}

// PayPetal refunds the escrowed amount plus our merchantCharge.
func TestRefund_RecordsFullAmountReturned(t *testing.T) {
	f := newEscrowFixture(t)
	id, ref := f.fundedMilestone(500)
	f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref}, http.StatusOK, nil)

	if m := f.milestone(id); m.Status != "refunded" {
		t.Fatalf("expected refunded, got %q", m.Status)
	}
	var txs []domain.Transaction
	f.get("/wallet/transactions", f.clientTok, http.StatusOK, &txs)
	var refund float64
	for _, tx := range txs {
		if tx.Type == "refund" {
			refund += tx.Amount
		}
	}
	if refund != 525 {
		t.Fatalf("expected a 525 refund (450 escrow + 75 fee), got %v", refund)
	}
	var summary domain.WalletSummary
	f.get("/wallet", f.clientTok, http.StatusOK, &summary)
	if summary.TotalSpent != 0 {
		t.Fatalf("expected nothing spent after a full refund, got %v", summary.TotalSpent)
	}
}

func TestFund_ReopeningUnpaidCheckoutKeepsOneRecord(t *testing.T) {
	f := newEscrowFixture(t)
	id := f.acceptedMilestone(500)
	if a, b := f.startFunding(id), f.startFunding(id); a != b {
		t.Fatalf("expected the same reference, got %q and %q", a, b)
	}
	pending, err := f.app.escrowRepo.ListPending(context.Background())
	if err != nil || len(pending) != 1 {
		t.Fatalf("expected one pending record, got %d (%v)", len(pending), err)
	}
}

// Customer IDs are per PayPetal environment; one stored from another must not
// block payments, and the payout account has to follow the user to the new
// customer or a paid-in escrow could never be paid out.
func TestFund_RecreatesUnknownCustomersWithPayoutAccount(t *testing.T) {
	f := newEscrowFixture(t)
	id := f.acceptedMilestone(500)
	before, _ := f.app.userRepo.GetByID(context.Background(), f.musicianID)
	f.app.paypetal.ForgetCustomers()

	ref := f.startFunding(id)
	agreement, err := f.app.escrowRepo.GetByReference(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if agreement.CounterpartyCustomerID == before.PayPetalCustomerID {
		t.Fatal("expected the talent's unknown customer to be replaced")
	}
	for _, customerID := range []string{agreement.InitiatorCustomerID, agreement.CounterpartyCustomerID} {
		if !f.app.paypetal.HasPayoutAccount(customerID) {
			t.Fatalf("%s has no payout account", customerID)
		}
	}
}

// If the account on file doesn't belong to the same holder in the current
// environment, nothing may be paid in until the talent adds one.
func TestFund_BlocksWhenPayoutAccountCannotBeCarriedOver(t *testing.T) {
	f := newEscrowFixture(t)
	id := f.acceptedMilestone(500)
	ctx := context.Background()
	musician, _ := f.app.userRepo.GetByID(ctx, f.musicianID)
	musician.PayoutAccount.AccountName = "Somebody Else"
	_ = f.app.userRepo.Update(ctx, musician)
	f.app.paypetal.ForgetCustomers()

	f.post("/milestones/fund", f.clientTok, map[string]any{"contract_id": f.contractID, "milestone_id": id}, http.StatusConflict, nil)

	// Adding it again unblocks.
	f.post("/payout-account", f.musicianTok, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0123456789",
	}, http.StatusOK, nil)
	f.startFunding(id)
}

// Relinking a payout account is also where most users with a stale ID turn up.
func TestPayoutAccount_LinkRecreatesUnknownCustomer(t *testing.T) {
	f := newEscrowFixture(t)
	f.app.paypetal.ForgetCustomers()
	f.post("/payout-account", f.musicianTok, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "9876543210",
	}, http.StatusOK, nil)
	u, _ := f.app.userRepo.GetByID(context.Background(), f.musicianID)
	if u.PayoutAccount == nil || u.PayoutAccount.AccountNumber != "9876543210" || !f.app.paypetal.HasPayoutAccount(u.PayPetalCustomerID) {
		t.Fatalf("expected the new account linked to a live customer, got %#v", u.PayoutAccount)
	}
}

// References are letters and digits only: live payments made against
// "ms:<id>" references were never matched by PayPetal.
func TestFund_ReferenceIsAlphanumeric(t *testing.T) {
	f := newEscrowFixture(t)
	id := f.acceptedMilestone(500)
	if ref := f.startFunding(id); ref != "ms"+id {
		t.Fatalf("expected ms%s, got %q", id, ref)
	}
}

// PayPetal may still apply a payment made against an old-format reference
// after the same milestone was reopened under the new one. Release has to act
// on the agreement that was actually paid.
func TestFinalizeFund_OldReferencePaidAfterReopening(t *testing.T) {
	f := newEscrowFixture(t)
	ctx := context.Background()
	id := f.acceptedMilestone(500)
	newRef, oldRef := f.startFunding(id), "ms:"+id

	agreement, err := f.app.escrowRepo.GetByReference(ctx, newRef)
	if err != nil {
		t.Fatal(err)
	}
	old := *agreement
	old.ID, old.ReferenceID = "", oldRef
	if err := f.app.escrowRepo.Create(ctx, &old); err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.paypetal.CreateTrustCoreAgreement(ctx, paypetal.CreateAgreementInput{
		ReferenceID: oldRef, CounterpartyCustomerID: old.CounterpartyCustomerID,
	}); err != nil {
		t.Fatal(err)
	}
	f.app.paypetal.SimulatePaymentCompleted(oldRef)

	f.post("/milestones/fund/finalize", f.clientTok, map[string]any{"reference": oldRef}, http.StatusOK, nil)
	f.post("/milestones/release", f.clientTok, map[string]any{"contract_id": f.contractID, "milestone_id": id}, http.StatusOK, nil)

	paid, _ := f.app.paypetal.GetTrustCoreAgreement(ctx, oldRef)
	unpaid, _ := f.app.paypetal.GetTrustCoreAgreement(ctx, newRef)
	if paid.PayoutStatus == "NONE" || unpaid.PayoutStatus != "NONE" {
		t.Fatalf("released the wrong agreement: old=%s new=%s", paid.PayoutStatus, unpaid.PayoutStatus)
	}
}
