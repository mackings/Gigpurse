package http_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/repository/memory"
)

const testPayoutWallet = "gigpurse-payouts"

// The talent's share of a ₦500 milestone, and what the sub-wallet pays for it.
const talentShare = 450.0

func newAutoFixture(t *testing.T) *escrowFixture {
	t.Helper()
	f := newEscrowFixtureOn(t, newTestAppWith(testPayoutWallet))
	var info domain.AutoPayoutInfo
	f.get("/admin/payouts/auto", f.admTok, http.StatusOK, &info) // creates the sub-wallet
	if !info.Enabled || info.AccountNumber == "" {
		t.Fatalf("expected the payout sub-wallet to be set up, got %#v", info)
	}
	return f
}

// released funds a ₦500 milestone and has the client release it, returning
// the talent's payout and the PayPetal reference it's paid under.
func (f *escrowFixture) released() (payoutID, autoRef string) {
	f.t.Helper()
	id, ref := f.vaultFunded(500)
	f.sendCode(id)
	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusOK, nil)
	return ref + ":" + domain.PayoutTalent, payoutRef("pt", ref)
}

// payoutRef is the PayPetal reference a payout goes out under: letters and
// digits only (the in-memory IDs here have underscores; real ones are hex).
func payoutRef(prefix, ref string) string {
	return prefix + strings.ReplaceAll(ref, "_", "")
}

func (f *escrowFixture) payout(id string) domain.Payout {
	f.t.Helper()
	p, err := f.app.payoutRepo.GetByID(context.Background(), id)
	if err != nil {
		f.t.Fatalf("payout %s: %v", id, err)
	}
	return *p
}

func (f *escrowFixture) runPayouts() { f.app.payouts.ProcessAuto(context.Background()) }

// timePasses brings a payout's next automatic try forward to now.
func (f *escrowFixture) timePasses(id string) {
	f.t.Helper()
	p := f.payout(id)
	if p.RetryAt == nil {
		f.t.Fatalf("payout %s has no retry scheduled", id)
	}
	past := time.Now().Add(-time.Second)
	p.RetryAt = &past
	if err := f.app.payoutRepo.Update(context.Background(), &p); err != nil {
		f.t.Fatal(err)
	}
}

func (f *escrowFixture) customerOf(userID string) string {
	u, _ := f.app.userRepo.GetByID(context.Background(), userID)
	return u.PayPetalCustomerID
}

func TestAutoPayout_PaysTalentOnce(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	f.get("/admin/payouts/auto", f.musicianTok, http.StatusForbidden, nil)

	pid, ref := f.released()
	f.runPayouts()

	tc := pp.TrustCoreState(ref)
	talent := f.customerOf(f.musicianID)
	if tc == nil || tc.Amount != talentShare || tc.CustomerID != talent || tc.PayoutStatus != "PENDING" {
		t.Fatalf("expected a %v transfer to the talent under way, got %#v", talentShare, tc)
	}
	if got := pp.LinkedAccount(talent); got != "000/0123456789" {
		t.Fatalf("expected the talent's bank account linked on PayPetal, got %q", got)
	}
	if bal := pp.SubWalletBalance(); bal != 1000-talentShare-memory.PayoutFee {
		t.Fatalf("expected the sub-wallet debited once, balance %v", bal)
	}
	if p := f.payout(pid); p.Status != "sending" || p.AutoRef != ref {
		t.Fatalf("expected sending under %s, got %#v", ref, p)
	}
	// An admin can't pay it by hand on top.
	f.post("/admin/payouts/mark-paid", f.admTok, map[string]any{"id": pid, "note": "NIP-1"}, http.StatusBadRequest, nil)

	f.runPayouts()
	f.runPayouts()
	if pp.TrustCoreCount() != 1 || pp.Completes != 1 {
		t.Fatalf("nothing may be repeated while the transfer is in flight: %d agreements, %d transfers", pp.TrustCoreCount(), pp.Completes)
	}
	if w := f.wallet(f.musicianTok); w.TotalEarned != 0 {
		t.Fatalf("not earned until the transfer lands, got %v", w.TotalEarned)
	}

	pp.SettleTransfer(ref, "COMPLETED")
	f.post("/webhooks/paypetal", "", map[string]any{"event": "trustcore.payment.completed", "reference": ref}, http.StatusOK, nil)
	f.runPayouts()
	p := f.payout(pid)
	if p.Status != "paid" || p.PaidBy != "paypetal" || p.Note != "PayPetal "+ref || p.PaidAt == nil {
		t.Fatalf("expected paid by PayPetal, got %#v", p)
	}
	f.runPayouts()
	if w := f.wallet(f.musicianTok); w.TotalEarned != talentShare {
		t.Fatalf("expected %v earned once, got %v", talentShare, w.TotalEarned)
	}
	if pp.TrustCoreCount() != 1 || pp.Completes != 1 {
		t.Fatalf("paid twice: %d agreements, %d transfers", pp.TrustCoreCount(), pp.Completes)
	}
}

func TestAutoPayout_GoesOutAsSoonAsTheWalletIsToppedUp(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pid, ref := f.released()
	f.runPayouts()

	p := f.payout(pid)
	if p.Status != "due" || p.WaitingOn != "funds" || !strings.Contains(p.AutoError, "topped up") {
		t.Fatalf("expected it waiting for a top-up, got %#v", p)
	}
	if pp.TrustCoreState(ref) != nil {
		t.Fatal("nothing should be set aside without the balance")
	}
	f.runPayouts()
	f.runPayouts()
	if pp.Creates != 1 {
		t.Fatalf("it shouldn't keep asking PayPetal while the balance hasn't moved, got %d attempts", pp.Creates)
	}

	// No admin: the next pass after the top-up sends it.
	pp.TopUp(talentShare + memory.PayoutFee)
	f.runPayouts()
	if p := f.payout(pid); p.Status != "sending" || p.AutoError != "" || pp.TrustCoreState(ref).PayoutStatus != "PENDING" || pp.SubWalletBalance() != 0 {
		t.Fatalf("expected it sent on the first pass after the top-up, got %#v / %#v", p, pp.TrustCoreState(ref))
	}
	pp.SettleTransfer(ref, "COMPLETED")
	f.runPayouts()
	if p := f.payout(pid); p.Status != "paid" {
		t.Fatalf("expected paid, got %#v", p)
	}
}

func TestAutoPayout_FailedTransfersRetryOnTheirOwn(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	pid, ref := f.released()
	f.runPayouts()
	for i := 1; i <= 8; i++ { // well past any cap: it never gives up
		pp.SettleTransfer(ref, "FAILED")
		f.runPayouts()
		p := f.payout(pid)
		if pp.Completes != i || p.Status != "sending" || p.WaitingOn != "transfer" || p.RetryAt == nil || p.Held {
			t.Fatalf("failure %d: expected a retry scheduled, not sent at once: %d transfers, %#v", i, pp.Completes, p)
		}
		if gap := time.Until(*p.RetryAt); gap > retryGapFor(i) || gap < retryGapFor(i)-time.Minute {
			t.Fatalf("failure %d: retry in %v, want about %v", i, gap, retryGapFor(i))
		}
		f.runPayouts() // too early: nothing happens
		if pp.Completes != i {
			t.Fatalf("failure %d: resent before its retry time", i)
		}
		f.timePasses(pid)
		f.runPayouts()
		if pp.Completes != i+1 {
			t.Fatalf("failure %d: expected it resent automatically, got %d transfers", i, pp.Completes)
		}
	}
	pp.SettleTransfer(ref, "COMPLETED")
	f.runPayouts()
	if p := f.payout(pid); p.Status != "paid" || pp.TrustCoreCount() != 1 || pp.SubWalletBalance() != 1000-talentShare-memory.PayoutFee {
		t.Fatalf("expected one agreement paid once, got %#v, %d agreements, balance %v", p, pp.TrustCoreCount(), pp.SubWalletBalance())
	}
	if w := f.wallet(f.musicianTok); w.TotalEarned != talentShare {
		t.Fatalf("expected %v earned once, got %v", talentShare, w.TotalEarned)
	}
}

// retryGapFor mirrors the engine's schedule: 1m, 5m, 15m, 30m, 1h, then 3h.
func retryGapFor(failures int) time.Duration {
	gaps := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour, 3 * time.Hour}
	if failures > len(gaps) {
		failures = len(gaps)
	}
	return gaps[failures-1]
}

func TestAutoPayout_GoesOutWhenTheTalentFixesTheirBank(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	pid, ref := f.released()
	pp.RejectAccount = "0123456789" // the account on file stops verifying
	f.runPayouts()
	p := f.payout(pid)
	if p.Status != "due" || p.WaitingOn != "account" || pp.TrustCoreState(ref) != nil {
		t.Fatalf("expected it waiting on the talent's bank account with nothing set aside, got %#v", p)
	}
	f.runPayouts()
	var notes []domain.Notification
	f.get("/notifications", f.musicianTok, http.StatusOK, &notes)
	asked := 0
	for _, n := range notes {
		if n.Title == "Update your bank account" {
			asked++
		}
	}
	if asked != 1 {
		t.Fatalf("expected the talent asked once to update their bank account, got %d", asked)
	}

	f.post("/payout-account", f.musicianTok, map[string]any{"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0222222222"}, http.StatusOK, nil)
	f.runPayouts()
	tc := pp.TrustCoreState(ref)
	if tc == nil || tc.PayoutStatus != "PENDING" || pp.LinkedAccount(f.customerOf(f.musicianID)) != "000/0222222222" {
		t.Fatalf("expected it sent to the new account straight away, got %#v", tc)
	}
}

func TestAutoPayout_LostResponseFundsOnce(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	pp.LoseCreateResponse = true
	pid, _ := f.released()
	f.runPayouts()
	if p := f.payout(pid); p.Status != "sending" || pp.Completes != 0 {
		t.Fatalf("an unconfirmed funding must stay sending, not go back to due: %#v", p)
	}
	f.runPayouts()
	if pp.TrustCoreCount() != 1 || pp.Completes != 1 || pp.SubWalletBalance() != 1000-talentShare-memory.PayoutFee {
		t.Fatalf("expected it found and sent once: %d agreements, %d transfers, balance %v", pp.TrustCoreCount(), pp.Completes, pp.SubWalletBalance())
	}
}

func TestAutoPayout_PayByHandReturnsTheMoney(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	pid, ref := f.released()
	f.runPayouts()

	f.post("/admin/payouts/pay-by-hand", f.admTok, map[string]any{"id": pid}, http.StatusBadRequest, nil) // transfer in flight
	pp.SettleTransfer(ref, "FAILED")
	var got domain.Payout
	f.post("/admin/payouts/pay-by-hand", f.admTok, map[string]any{"id": pid}, http.StatusOK, &got)
	if got.Status != "due" || !got.ManualOnly || pp.TrustCoreState(ref).RefundStatus != "COMPLETED" {
		t.Fatalf("expected the money back in the sub-wallet and the payout due, got %#v", got)
	}
	if bal := pp.SubWalletBalance(); bal != 1000-memory.PayoutFee {
		t.Fatalf("expected everything but PayPetal's fee back, balance %v", bal)
	}
	f.runPayouts()
	if pp.Completes != 1 {
		t.Fatalf("a payout taken over by hand must be left alone, got %d transfers", pp.Completes)
	}
	f.post("/admin/payouts/mark-paid", f.admTok, map[string]any{"id": pid, "note": "NIP-9"}, http.StatusOK, nil)
	if w := f.wallet(f.musicianTok); w.TotalEarned != talentShare {
		t.Fatalf("expected %v earned, got %v", talentShare, w.TotalEarned)
	}
}

func TestAutoPayout_RetryAfterPayByHandUsesANewReference(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	pid, ref := f.released()
	f.runPayouts()
	pp.SettleTransfer(ref, "FAILED")
	f.post("/admin/payouts/pay-by-hand", f.admTok, map[string]any{"id": pid}, http.StatusOK, nil)

	var got domain.Payout
	f.post("/admin/payouts/retry", f.admTok, map[string]any{"id": pid}, http.StatusOK, &got)
	if got.Status != "sending" || got.AutoRef != ref+"n1" || pp.TrustCoreState(ref+"n1") == nil {
		t.Fatalf("expected a fresh agreement %sn1, got %#v", ref, got)
	}
	pp.SettleTransfer(ref+"n1", "COMPLETED")
	f.runPayouts()
	if p := f.payout(pid); p.Status != "paid" {
		t.Fatalf("expected paid, got %#v", p)
	}
	if w := f.wallet(f.musicianTok); w.TotalEarned != talentShare {
		t.Fatalf("expected %v earned once, got %v", talentShare, w.TotalEarned)
	}
}

func TestAutoPayout_NeverReleasesSomeoneElsesAgreement(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	id, ref := f.vaultFunded(500)
	pp.SeedTrustCore(payoutRef("pt", ref), 999, "cust_someone_else")
	f.sendCode(id)
	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusOK, nil)
	pid := ref + ":" + domain.PayoutTalent

	f.runPayouts()
	f.runPayouts()
	if p := f.payout(pid); pp.Completes != 0 || p.Status != "sending" || !p.Held || !strings.Contains(p.AutoError, "Check it in the PayPetal dashboard") {
		t.Fatalf("expected it held for an admin, got %d transfers, %#v", pp.Completes, p)
	}
	var got domain.Payout
	f.post("/admin/payouts/pay-by-hand", f.admTok, map[string]any{"id": pid}, http.StatusOK, &got)
	if got.Status != "due" || !got.ManualOnly || pp.TrustCoreState(payoutRef("pt", ref)).RefundStatus != "NONE" {
		t.Fatalf("expected it handed over without touching the other agreement, got %#v / %#v", got, pp.TrustCoreState(payoutRef("pt", ref)))
	}
}

func TestAutoPayout_ConcurrentPassesSendOnce(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	_, ref := f.released()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); f.runPayouts() }()
	}
	wg.Wait()
	f.runPayouts()
	if pp.TrustCoreCount() != 1 || pp.Completes != 1 || pp.TrustCoreState(ref) == nil || pp.SubWalletBalance() != 1000-talentShare-memory.PayoutFee {
		t.Fatalf("expected one agreement sent once: %d agreements, %d transfers, balance %v", pp.TrustCoreCount(), pp.Completes, pp.SubWalletBalance())
	}
}

func TestAutoPayout_RefundsTheClient(t *testing.T) {
	f := newAutoFixture(t)
	pp := f.app.paypetal
	pp.TopUp(1000)
	_, ref := f.vaultFunded(500)
	f.post("/wallet/escrow/code", f.clientTok, map[string]any{"reference_id": ref}, http.StatusOK, nil)
	f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref, "code": memory.VaultCode}, http.StatusOK, nil)
	f.runPayouts()
	tc := pp.TrustCoreState(payoutRef("pr", ref))
	if tc == nil || tc.Amount != 525 || tc.CustomerID != f.customerOf(f.clientID) {
		t.Fatalf("expected ₦525 on its way to the client, got %#v", tc)
	}
	pp.SettleTransfer(payoutRef("pr", ref), "COMPLETED")
	f.runPayouts()
	if p := f.payout(ref + ":" + domain.PayoutRefund); p.Status != "paid" || p.Kind != domain.PayoutRefund {
		t.Fatalf("expected the refund paid, got %#v", p)
	}
}

func TestAutoPayout_OffLeavesPayoutsForAdmins(t *testing.T) {
	f := newEscrowFixture(t)
	var info domain.AutoPayoutInfo
	f.get("/admin/payouts/auto", f.admTok, http.StatusOK, &info)
	if info.Enabled {
		t.Fatal("automatic payouts should be off without a sub-wallet")
	}
	pid, _ := f.released()
	f.runPayouts()
	if p := f.payout(pid); p.Status != "due" || f.app.paypetal.Creates != 0 {
		t.Fatalf("expected it left due for an admin, got %#v", p)
	}
	f.post("/admin/payouts/retry", f.admTok, map[string]any{"id": pid}, http.StatusBadRequest, nil)
}
