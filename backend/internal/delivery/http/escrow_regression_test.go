package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gigpurse/internal/domain"
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
	f.app.paypetal.SimulateVaultPaid(reference)
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

	if m := f.milestone(second); m.Status != "settling" {
		t.Fatalf("expected the second milestone ruled on, got %q", m.Status)
	}
	if m := f.milestone(first); m.Status != "disputed" {
		t.Fatalf("expected the first milestone still under its own dispute, got %q", m.Status)
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

// References are letters and digits only: live payments made against
// "ms:<id>" references were never matched by PayPetal.
func TestFund_ReferenceIsAlphanumeric(t *testing.T) {
	f := newEscrowFixture(t)
	id := f.acceptedMilestone(500)
	if ref := f.startFunding(id); ref != "ms"+id {
		t.Fatalf("expected ms%s, got %q", id, ref)
	}
}

// Customer IDs are per PayPetal environment; one stored from another must not
// block payments.
func TestFund_RecreatesUnknownCustomer(t *testing.T) {
	f := newEscrowFixture(t)
	f.startFunding(f.acceptedMilestone(500)) // first payment creates the customer
	before, _ := f.app.userRepo.GetByID(context.Background(), f.clientID)
	f.app.paypetal.ForgetCustomers()
	f.startFunding(f.acceptedMilestone(500))
	after, _ := f.app.userRepo.GetByID(context.Background(), f.clientID)
	if before.PayPetalCustomerID == "" || after.PayPetalCustomerID == before.PayPetalCustomerID || f.app.paypetal.LastVault.CustomerID != after.PayPetalCustomerID {
		t.Fatalf("expected a new customer, had %q now %q", before.PayPetalCustomerID, after.PayPetalCustomerID)
	}
}

// Payouts are made by hand to the account on file, so one PayPetal can't
// resolve has to stop a payment before money goes in — without throwing the
// details away, since the lookup may be at fault.
func TestFund_BlocksOnDeadBankAccount(t *testing.T) {
	f := newEscrowFixture(t)
	id := f.acceptedMilestone(500)
	body := map[string]any{"contract_id": f.contractID, "milestone_id": id}
	f.app.paypetal.RejectAccount = "0123456789"
	f.post("/milestones/fund", f.clientTok, body, http.StatusConflict, nil)
	if u, _ := f.app.userRepo.GetByID(context.Background(), f.musicianID); u.PayoutAccount == nil {
		t.Fatal("the saved account should be kept")
	}
	for _, tok := range []string{f.musicianTok, f.clientTok} {
		f.post("/payout-account", tok, map[string]any{"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "5555555555"}, http.StatusOK, nil)
	}
	f.startFunding(id)
}
