package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/repository/memory"
)

func (f *escrowFixture) vaultFunded(amount float64) (id, reference string) {
	f.t.Helper()
	id = f.acceptedMilestone(amount)
	reference = f.startFunding(id)
	f.app.paypetal.SimulateVaultPaid(reference)
	f.post("/webhooks/paypetal", "", map[string]any{"event": "trustvault.funded", "reference": reference}, http.StatusOK, nil)
	if m := f.milestone(id); m.Status != "funded" || m.EscrowProvider != "trustvault" {
		f.t.Fatalf("expected a funded trustvault milestone, got %q/%q", m.Status, m.EscrowProvider)
	}
	return id, reference
}

func (f *escrowFixture) body(id string, extra ...any) map[string]any {
	b := map[string]any{"contract_id": f.contractID, "milestone_id": id}
	for i := 0; i+1 < len(extra); i += 2 {
		b[extra[i].(string)] = extra[i+1]
	}
	return b
}

func (f *escrowFixture) sendCode(id string) {
	f.post("/milestones/release/code", f.clientTok, f.body(id), http.StatusOK, nil)
}

func (f *escrowFixture) payouts(status string) map[string]domain.Payout {
	f.t.Helper()
	var ps []domain.Payout
	f.get("/admin/payouts?status="+status, f.admTok, http.StatusOK, &ps)
	out := map[string]domain.Payout{}
	for _, p := range ps {
		out[p.Kind] = p
	}
	return out
}

func (f *escrowFixture) wallet(tok string) (w domain.WalletSummary) {
	f.get("/wallet", tok, http.StatusOK, &w)
	return
}

func TestVault_FundReleasePayout(t *testing.T) {
	f := newEscrowFixture(t)
	id, ref := f.vaultFunded(500)

	in := f.app.paypetal.LastVault
	if in.Amount != "525.00" || in.ReferenceID != "ms"+id || in.CustomerID == "" {
		t.Fatalf("unexpected TrustVault request: %#v", in)
	}

	// Nothing moves without a code.
	f.post("/milestones/release", f.clientTok, f.body(id), http.StatusConflict, nil)
	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusBadRequest, nil) // never requested
	f.post("/milestones/release/code", f.musicianTok, f.body(id), http.StatusBadRequest, nil)                    // not the payer
	f.sendCode(id)
	f.post("/milestones/release", f.musicianTok, f.body(id, "code", memory.VaultCode), http.StatusBadRequest, nil)
	f.post("/milestones/release", f.clientTok, f.body(id, "code", "000000"), http.StatusBadRequest, nil)
	if m := f.milestone(id); m.Status != "funded" {
		t.Fatalf("a wrong code must leave it funded, got %q", m.Status)
	}
	if n := len(f.payouts("pending")) + len(f.payouts("due")); n != 0 {
		t.Fatalf("a wrong code must not leave payouts behind, got %d", n)
	}

	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusOK, nil)
	if m := f.milestone(id); m.Status != "released" {
		t.Fatalf("expected released, got %q", m.Status)
	}
	due := f.payouts("due")
	p := due[domain.PayoutTalent]
	if len(due) != 1 || p.Amount != 450 || p.UserID != f.musicianID || p.Account == nil || p.Account.AccountNumber != "0123456789" {
		t.Fatalf("expected 450 due to the talent with bank details, got %#v", due)
	}
	// Not paid until an admin says so.
	if w := f.wallet(f.musicianTok); w.TotalEarned != 0 {
		t.Fatalf("talent shouldn't show earnings yet, got %v", w.TotalEarned)
	}

	f.get("/admin/payouts", f.musicianTok, http.StatusForbidden, nil)
	f.post("/admin/payouts/mark-paid", f.clientTok, map[string]any{"id": p.ID, "note": "x"}, http.StatusForbidden, nil)
	f.post("/admin/payouts/mark-paid", f.admTok, map[string]any{"id": p.ID, "note": " "}, http.StatusBadRequest, nil)
	f.post("/admin/payouts/mark-paid", f.admTok, map[string]any{"id": p.ID, "note": "NIP-1"}, http.StatusOK, nil)
	f.post("/admin/payouts/mark-paid", f.admTok, map[string]any{"id": p.ID, "note": "NIP-1"}, http.StatusBadRequest, nil)
	if w := f.wallet(f.musicianTok); w.TotalEarned != 450 {
		t.Fatalf("expected 450 earned once, got %v", w.TotalEarned)
	}
	if w := f.wallet(f.clientTok); w.TotalSpent != 525 {
		t.Fatalf("expected the client to have spent 525, got %v", w.TotalSpent)
	}
	if paid := f.payouts("paid")[domain.PayoutTalent]; paid.Note != "NIP-1" || paid.PaidAt == nil {
		t.Fatalf("payout record incomplete: %#v", paid)
	}
	a, _ := f.app.escrowRepo.GetByReference(context.Background(), ref)
	if a.PayoutStatus != "COMPLETED" {
		t.Fatalf("expected payout completed on the agreement, got %q", a.PayoutStatus)
	}
	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusBadRequest, nil)
}

func TestVault_ClientRefundReturnsEverything(t *testing.T) {
	f := newEscrowFixture(t)
	id, ref := f.vaultFunded(500)
	f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref}, http.StatusConflict, nil)
	f.post("/wallet/escrow/code", f.clientTok, map[string]any{"reference_id": ref}, http.StatusOK, nil)
	f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref, "code": memory.VaultCode}, http.StatusOK, nil)
	if m := f.milestone(id); m.Status != "refunded" {
		t.Fatalf("expected refunded, got %q", m.Status)
	}
	due := f.payouts("due")
	if len(due) != 1 || due[domain.PayoutRefund].Amount != 525 {
		t.Fatalf("expected a 525 refund due, got %#v", due)
	}
	f.post("/admin/payouts/mark-paid", f.admTok, map[string]any{"id": due[domain.PayoutRefund].ID, "note": "R1"}, http.StatusOK, nil)
	if w := f.wallet(f.clientTok); w.TotalSpent != 0 {
		t.Fatalf("expected nothing spent, got %v", w.TotalSpent)
	}
}

func TestVault_DisputeRulings(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		award, talent, refund float64
		status                string
	}{
		{"talent wins", 450, 450, 0, "released"},
		{"client wins", 0, 0, 525, "refunded"},
		{"split", 200, 200, 325, "released"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newEscrowFixture(t)
			id, ref := f.vaultFunded(500)
			var d domain.Dispute
			f.post("/disputes", f.musicianTok, map[string]any{"contract_id": f.contractID, "reason": "Unpaid"}, http.StatusCreated, &d)
			// Locked: no code, no release, no refund.
			f.post("/milestones/release/code", f.clientTok, f.body(id), http.StatusBadRequest, nil)
			f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusBadRequest, nil)
			f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref, "code": memory.VaultCode}, http.StatusBadRequest, nil)

			var holdings []domain.EscrowHolding
			f.get("/wallet/escrow", f.clientTok, http.StatusOK, &holdings)
			if len(holdings) != 1 || !holdings[0].Disputed {
				t.Fatalf("expected the holding flagged as disputed, got %#v", holdings)
			}
			f.post("/disputes/join", f.admTok, map[string]any{"dispute_id": d.ID}, http.StatusOK, nil)
			f.post("/admin/disputes/resolve", f.admTok, map[string]any{"dispute_id": d.ID, "resolution": "x", "talent_amount": 451}, http.StatusBadRequest, nil)
			f.post("/admin/disputes/resolve", f.admTok, map[string]any{"dispute_id": d.ID, "resolution": "x", "talent_amount": tc.award}, http.StatusOK, nil)
			m := f.milestone(id)
			if m.Status != "settling" || m.SettlementTalent == nil || *m.SettlementTalent != tc.award {
				t.Fatalf("expected settling with the ruling, got %#v", m)
			}
			// The client can't take it back through the refund route...
			f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref, "code": memory.VaultCode}, http.StatusBadRequest, nil)
			// ...or start something new until they confirm.
			f.post("/jobs", f.clientTok, map[string]any{"title": "t", "description": "d", "instrument": "Guitar", "genre": "x", "location": "L", "budget": 100,
				"experience_level": "intermediate", "duration": "less_than_1_week", "project_type": "one_time"}, http.StatusConflict, nil)

			f.sendCode(id)
			f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusOK, nil)
			if got := f.milestone(id).Status; got != tc.status {
				t.Fatalf("expected %q, got %q", tc.status, got)
			}
			due := f.payouts("due")
			if due[domain.PayoutTalent].Amount != tc.talent || due[domain.PayoutRefund].Amount != tc.refund {
				t.Fatalf("expected talent %v / refund %v, got %#v", tc.talent, tc.refund, due)
			}
		})
	}
}

// A release whose response was lost must end up right either way.
func TestVault_LostReleaseResponse(t *testing.T) {
	ctx := context.Background()
	age := func(f *escrowFixture) {
		pending, _ := f.app.payoutRepo.ListByStatus(ctx, "pending")
		for _, p := range pending {
			p.CreatedAt = time.Now().Add(-5 * time.Minute)
			_ = f.app.payoutRepo.Update(ctx, p)
		}
	}
	for _, applied := range []bool{true, false} {
		f := newEscrowFixture(t)
		id, ref := f.vaultFunded(500)
		f.sendCode(id)
		f.app.paypetal.FailNextVaultRelease(ref, applied)
		f.app.paypetal.SetVaultDown(true)
		f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusBadRequest, nil)
		if len(f.payouts("due")) != 0 || len(f.payouts("pending")) != 1 {
			t.Fatal("an unknown outcome must stay pending")
		}
		age(f)
		f.app.milestones.ReconcileVault(ctx) // still can't tell
		if len(f.payouts("pending")) != 1 {
			t.Fatal("reconciler acted without knowing")
		}
		f.app.paypetal.SetVaultDown(false)
		f.app.milestones.ReconcileVault(ctx)
		f.app.milestones.ReconcileVault(ctx)
		want, due := "funded", 0
		if applied {
			want, due = "released", 1
		}
		if got := f.milestone(id).Status; got != want || len(f.payouts("due")) != due || len(f.payouts("pending")) != 0 {
			t.Fatalf("applied=%v: status %q, due %d", applied, got, len(f.payouts("due")))
		}
		if !applied { // and it can be released normally afterwards
			f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusOK, nil)
			if len(f.payouts("due")) != 1 {
				t.Fatal("retry didn't produce a payout")
			}
		}
	}

	// Response lost but PayPetal reachable: resolved on the spot.
	f := newEscrowFixture(t)
	id, ref := f.vaultFunded(500)
	f.sendCode(id)
	f.app.paypetal.FailNextVaultRelease(ref, true)
	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusOK, nil)
	if len(f.payouts("due")) != 1 {
		t.Fatal("expected one payout due")
	}
}

// Records from before TrustVault are inert: nothing is sent to PayPetal for
// them and no money can move.
func TestVault_OldRecordsAreInert(t *testing.T) {
	f := newEscrowFixture(t)
	ctx := context.Background()
	id, ref := f.vaultFunded(500)
	a, _ := f.app.escrowRepo.GetByReference(ctx, ref)
	a.Provider = ""
	_ = f.app.escrowRepo.Update(ctx, a)
	f.post("/milestones/release/code", f.clientTok, f.body(id), http.StatusBadRequest, nil)
	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusBadRequest, nil)
	f.post("/wallet/escrow/refund", f.clientTok, map[string]any{"reference_id": ref, "code": memory.VaultCode}, http.StatusBadRequest, nil)
	f.post("/webhooks/paypetal", "", map[string]any{"reference": ref}, http.StatusOK, nil)
	var holdings []domain.EscrowHolding
	f.get("/wallet/escrow", f.clientTok, http.StatusOK, &holdings)
	if len(holdings) != 0 || len(f.payouts("due")) != 0 {
		t.Fatalf("old record should be inert, got %#v", holdings)
	}
}

// The webhook carries PayPetal's reference in "reference" for TrustVault
// releases and ours in "idempotencyKey".
func TestVault_WebhookUsesIdempotencyKey(t *testing.T) {
	f := newEscrowFixture(t)
	id := f.acceptedMilestone(500)
	ref := f.startFunding(id)
	f.app.paypetal.SimulateVaultPaid(ref)
	f.post("/webhooks/paypetal", "", map[string]any{"event": "trustvault.x", "reference": "26112418341347", "idempotencyKey": ref}, http.StatusOK, nil)
	if m := f.milestone(id); m.Status != "funded" {
		t.Fatalf("expected funded, got %q", m.Status)
	}
}

// Completing a contract moves no money, so it's blocked while any is held.
func TestCompleteContract_BlockedWhileMoneyHeld(t *testing.T) {
	f := newEscrowFixture(t)
	complete := map[string]any{"contract_id": f.contractID}
	f.acceptedMilestone(300) // agreed but unpaid: doesn't block
	id, _ := f.vaultFunded(500)
	f.post("/contracts/complete", f.clientTok, complete, http.StatusConflict, nil)

	var d domain.Dispute
	f.post("/disputes", f.musicianTok, map[string]any{"contract_id": f.contractID, "reason": "x"}, http.StatusCreated, &d)
	f.post("/contracts/complete", f.clientTok, complete, http.StatusConflict, nil) // disputed
	f.post("/disputes/join", f.admTok, map[string]any{"dispute_id": d.ID}, http.StatusOK, nil)
	f.post("/admin/disputes/resolve", f.admTok, map[string]any{"dispute_id": d.ID, "resolution": "x", "talent_amount": 450}, http.StatusOK, nil)
	f.post("/contracts/complete", f.clientTok, complete, http.StatusConflict, nil) // settling

	f.sendCode(id)
	f.post("/milestones/release", f.clientTok, f.body(id, "code", memory.VaultCode), http.StatusOK, nil)
	f.post("/contracts/complete", f.clientTok, complete, http.StatusOK, nil)
}
