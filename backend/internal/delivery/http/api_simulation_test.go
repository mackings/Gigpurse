package http_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	delivery "gigpurse/internal/delivery/http"
	"gigpurse/internal/domain"
	"gigpurse/internal/repository/memory"
	"gigpurse/internal/usecase"

	"github.com/gorilla/websocket"
)

func TestSimulateClientMusicianAPIFlow(t *testing.T) {
	t.Setenv("JWT_SECRET", "api-simulation-secret")
	t.Setenv("ALLOW_ADMIN_SIGNUP", "true")

	app := newTestApp()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	client := &apiClient{t: t, baseURL: server.URL, http: server.Client()}
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	clientUser := client.signup("client@example.com", "password123", "client", "Demo Client")
	musicianUser := client.signup("musician@example.com", "password123", "musician", "Demo Musician")
	adminUser := client.signup("admin@example.com", "password123", "admin", "Demo Admin")

	client.verifyEmail(app, clientUser.ID, "client@example.com", "111111")
	client.verifyEmail(app, musicianUser.ID, "musician@example.com", "222222")
	client.verifyEmail(app, adminUser.ID, "admin@example.com", "333333")

	clientToken := client.login("client@example.com", "password123")
	musicianToken := client.login("musician@example.com", "password123")
	adminToken := client.login("admin@example.com", "password123")

	client.post("/auth/password-reset/request", "", map[string]any{"email": "client@example.com"}, http.StatusOK, nil)
	resetToken := "known-reset-token"
	resetHash := sha256.Sum256([]byte(resetToken))
	err := app.resetRepo.Create(context.Background(), &domain.PasswordResetToken{
		UserID:    clientUser.ID,
		TokenHash: hex.EncodeToString(resetHash[:]),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed reset token: %v", err)
	}
	client.post("/auth/password-reset/confirm", "", map[string]any{"token": resetToken, "new_password": "password123"}, http.StatusOK, nil)

	client.get("/users/profile", clientToken, http.StatusOK, nil)
	client.put("/users/profile", clientToken, map[string]any{
		"name":     "Demo Client",
		"bio":      "Looking for reliable session musicians",
		"location": "Lagos",
		"client_profile": map[string]any{
			"company_name": "Gigpurse Events",
		},
	}, http.StatusOK, nil)
	client.put("/users/profile", musicianToken, map[string]any{
		"name":     "Demo Musician",
		"bio":      "Guitarist and producer",
		"location": "Lagos",
		"musician_profile": map[string]any{
			"stage_name":       "Demo Strings",
			"instruments":      []string{"Guitar"},
			"genres":           []string{"Afrobeats"},
			"experience_years": 7,
			"portfolio": []map[string]any{{
				"title":       "Live Session",
				"description": "Recorded guitar session",
				"url":         "https://example.com/session.mp4",
			}},
		},
	}, http.StatusOK, nil)

	var musicians []domain.User
	client.get("/musicians?genre=Afrobeats&instrument=Guitar&location=Lagos&min_exp=3&sort_by=experience", clientToken, http.StatusOK, &musicians)
	if len(musicians) != 1 || musicians[0].ID != musicianUser.ID {
		t.Fatalf("expected musician search to return the dummy musician, got %#v", musicians)
	}

	var musicianByID domain.User
	client.get("/musicians/"+musicianUser.ID, "", http.StatusOK, &musicianByID)
	if musicianByID.ID != musicianUser.ID {
		t.Fatalf("expected public musician lookup to return %q, got %#v", musicianUser.ID, musicianByID)
	}

	// A client needs a payout account on file before posting/hiring too —
	// it's the only place PayPetal can send money if a dispute later refunds
	// them. Set up ahead of the first job post, same as the musician's below.
	client.post("/payout-account/validate", clientToken, map[string]any{
		"bank_code": "000", "account_number": "0000000000",
	}, http.StatusOK, nil)
	client.post("/payout-account", clientToken, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0000000000",
	}, http.StatusOK, nil)

	var job domain.Job
	client.post("/jobs", clientToken, map[string]any{
		"title":            "Afrobeats guitar session",
		"description":      "Need guitar for a studio session",
		"instrument":       "Guitar",
		"genre":            "Afrobeats",
		"location":         "Lagos",
		"budget":           500,
		"experience_level": "intermediate",
		"duration":         "less_than_1_week",
		"project_type":     "one_time",
		"skills":           []string{"Guitar", "Session recording"},
	}, http.StatusCreated, &job)
	if job.Status != "pending_funding" {
		t.Fatalf("expected new job to start pending_funding, got %q", job.Status)
	}

	// A job stays invisible to talent until it's published — no money moves
	// here anymore (PayPetal needs both parties to create an agreement, and
	// no musician is known yet); real payment happens at hire time below.
	var fundedJob domain.Job
	client.post("/jobs/fund", clientToken, map[string]any{"job_id": job.ID}, http.StatusOK, &fundedJob)
	if fundedJob.Status != "open" || fundedJob.EscrowFunded {
		t.Fatalf("expected job to be open and not yet escrow-funded after publishing, got %#v", fundedJob)
	}
	job = fundedJob

	client.get("/jobs?id="+job.ID, clientToken, http.StatusOK, nil)
	client.get("/jobs?status=open&genre=Afrobeats&sort_by=budget&max_applications=5", musicianToken, http.StatusOK, nil)
	client.get("/jobs/recommended?limit=5", musicianToken, http.StatusOK, nil)

	var clientJobs []domain.Job
	client.get("/jobs?client_id="+clientUser.ID, clientToken, http.StatusOK, &clientJobs)
	if len(clientJobs) != 1 || clientJobs[0].ID != job.ID {
		t.Fatalf("expected client_id filter to return only the client's own job, got %#v", clientJobs)
	}

	var application domain.JobApplication
	client.post("/jobs/apply", musicianToken, map[string]any{
		"job_id":    job.ID,
		"proposal":  "I can deliver a clean session. WhatsApp and Paypal should be filtered in chat.",
		"price_bid": 450,
	}, http.StatusCreated, &application)
	client.get("/jobs/applications?job_id="+job.ID, clientToken, http.StatusOK, nil)
	client.get("/jobs/applications", musicianToken, http.StatusOK, nil)
	client.get("/jobs/mine?status=pending", musicianToken, http.StatusOK, nil)

	var chatMsg domain.ChatMessage
	client.post("/chats", clientToken, map[string]any{
		"recv_id": musicianUser.ID,
		"content": "Please do not ask for Paypal or WhatsApp outside the platform.",
	}, http.StatusCreated, &chatMsg)
	if strings.Contains(strings.ToLower(chatMsg.Content), "paypal") || strings.Contains(strings.ToLower(chatMsg.Content), "whatsapp") {
		t.Fatalf("expected chat content to be filtered, got %q", chatMsg.Content)
	}
	client.get("/chats/history?user_id="+musicianUser.ID, clientToken, http.StatusOK, nil)
	client.get("/chats/recent", clientToken, http.StatusOK, nil)

	clientWS := dialWS(t, wsURL+"/ws?token="+clientToken)
	defer clientWS.Close()
	musicianWS := dialWS(t, wsURL+"/ws?token="+musicianToken)
	defer musicianWS.Close()
	if err := clientWS.WriteJSON(map[string]string{"recv_id": musicianUser.ID, "content": "Realtime hello"}); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}
	type wsEnvelope struct {
		Type string             `json:"type"`
		Data domain.ChatMessage `json:"data"`
	}
	var senderEnvelope wsEnvelope
	if err := clientWS.ReadJSON(&senderEnvelope); err != nil {
		t.Fatalf("read websocket echo: %v", err)
	}
	if senderEnvelope.Type != "chat_message" {
		t.Fatalf("expected chat_message envelope, got %q", senderEnvelope.Type)
	}
	var receiverEnvelope wsEnvelope
	if err := musicianWS.ReadJSON(&receiverEnvelope); err != nil {
		t.Fatalf("read websocket receiver message: %v", err)
	}
	if receiverEnvelope.Type != "chat_message" {
		t.Fatalf("expected chat_message envelope, got %q", receiverEnvelope.Type)
	}

	// Set up the musician's payout account ahead of time — not required to
	// accept the application anymore (that's free/synchronous now), but
	// still required later when a milestone actually gets funded.
	client.post("/payout-account/validate", musicianToken, map[string]any{
		"bank_code": "000", "account_number": "0000000000",
	}, http.StatusOK, nil)
	client.post("/payout-account", musicianToken, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0000000000",
	}, http.StatusOK, nil)

	// Accepting hires immediately, at no cost — every payment happens
	// through milestones proposed on the resulting contract.
	var hireResult domain.Contract
	client.post("/jobs/applications/accept", clientToken, map[string]any{"application_id": application.ID}, http.StatusOK, &hireResult)
	if hireResult.ID == "" {
		t.Fatal("expected AcceptApplication to return a contract")
	}

	client.get("/jobs/mine?status=active", musicianToken, http.StatusOK, nil)

	var contracts []domain.Contract
	client.get("/contracts", clientToken, http.StatusOK, &contracts)
	if len(contracts) == 0 || contracts[0].ID != hireResult.ID {
		t.Fatalf("expected accepted application to create the contract returned by AcceptApplication, got %#v vs %q", contracts, hireResult.ID)
	}
	client.get("/contracts?id="+contracts[0].ID, clientToken, http.StatusOK, nil)

	var directHire domain.DirectHireRequest
	client.post("/direct-hires", clientToken, map[string]any{
		"musician_id": musicianUser.ID,
		"title":       "Private acoustic set",
		"description": "Direct hire for a private event",
		"location":    "Lekki, Lagos",
		"price":       300,
	}, http.StatusCreated, &directHire)
	if directHire.ProposedBy != clientUser.ID {
		t.Fatalf("expected proposed_by to be the client, got %s", directHire.ProposedBy)
	}
	client.get("/direct-hires?status=pending", musicianToken, http.StatusOK, nil)
	client.get("/direct-hires?id="+directHire.ID, musicianToken, http.StatusOK, nil)

	// Musician can't accept/decline their own... wait, they didn't propose;
	// but they CAN counter instead of accepting outright.
	var countered domain.DirectHireRequest
	client.post("/direct-hires/counter", musicianToken, map[string]any{
		"request_id": directHire.ID,
		"price":      400,
	}, http.StatusOK, &countered)
	if countered.Price != 400 || countered.ProposedBy != musicianUser.ID || len(countered.History) != 2 {
		t.Fatalf("expected counter-offer to update price/proposer/history, got %+v", countered)
	}

	// Client can't accept their own outstanding counter; musician can't
	// immediately re-counter without the client responding first — confirm
	// the musician is blocked from accepting/countering their own offer.
	client.post("/direct-hires/counter", musicianToken, map[string]any{
		"request_id": directHire.ID,
		"price":      450,
	}, http.StatusBadRequest, nil)

	var acceptedDirectHire domain.DirectHireRequest
	client.post("/direct-hires/respond", clientToken, map[string]any{
		"request_id": directHire.ID,
		"decision":   "accepted",
	}, http.StatusOK, &acceptedDirectHire)
	if acceptedDirectHire.ContractID == "" {
		t.Fatal("expected accepted direct hire to create a contract")
	}
	if acceptedDirectHire.Price != 400 {
		t.Fatalf("expected contract to be created at the negotiated price 400, got %v", acceptedDirectHire.Price)
	}

	client.post("/contracts/complete", clientToken, map[string]any{"contract_id": contracts[0].ID}, http.StatusOK, nil)
	client.post("/contracts/complete", clientToken, map[string]any{"contract_id": acceptedDirectHire.ContractID}, http.StatusOK, nil)
	client.get("/jobs/mine?status=completed", musicianToken, http.StatusOK, nil)

	// Job-sourced contract: reviews carry a populated job_id.
	var jobReview domain.Review
	client.post("/reviews", clientToken, map[string]any{"contract_id": contracts[0].ID, "rating": 5, "comment": "Excellent work"}, http.StatusCreated, &jobReview)
	client.post("/reviews", musicianToken, map[string]any{"contract_id": contracts[0].ID, "rating": 5, "comment": "Great client"}, http.StatusCreated, nil)
	if jobReview.JobID != job.ID {
		t.Fatalf("expected job-sourced review to carry job_id %q, got %q", job.ID, jobReview.JobID)
	}

	// Direct-hire-sourced contract: this was impossible before reviews were generalized
	// to attach to a Contract instead of requiring a Job.
	var directHireReview domain.Review
	client.post("/reviews", clientToken, map[string]any{"contract_id": acceptedDirectHire.ContractID, "rating": 5, "comment": "Great direct-hire experience"}, http.StatusCreated, &directHireReview)
	if directHireReview.JobID != "" {
		t.Fatalf("expected direct-hire-sourced review to have an empty job_id, got %q", directHireReview.JobID)
	}

	client.get("/reviews?user_id="+musicianUser.ID, clientToken, http.StatusOK, nil)
	client.get("/reviews/average?user_id="+musicianUser.ID, clientToken, http.StatusOK, nil)

	var notifications []domain.Notification
	client.get("/notifications", musicianToken, http.StatusOK, &notifications)
	if len(notifications) > 0 {
		client.post("/notifications/read", musicianToken, map[string]any{"notification_id": notifications[0].ID}, http.StatusOK, nil)
	}

	client.get("/talent/dashboard", musicianToken, http.StatusOK, nil)

	var dispute domain.Dispute
	client.post("/disputes", clientToken, map[string]any{"contract_id": contracts[0].ID, "reason": "Need admin review"}, http.StatusCreated, &dispute)
	client.get("/disputes", musicianToken, http.StatusOK, nil)
	client.get("/admin/disputes?status=open", adminToken, http.StatusOK, nil)

	// Neither party can talk in the dispute room until a moderator/admin joins.
	client.post("/disputes/messages", clientToken, map[string]any{"dispute_id": dispute.ID, "content": "hello?"}, http.StatusBadRequest, nil)

	var joined domain.Dispute
	client.post("/disputes/join", adminToken, map[string]any{"dispute_id": dispute.ID}, http.StatusOK, &joined)
	if joined.ModeratorID != adminUser.ID {
		t.Fatalf("expected joining admin to be recorded as moderator, got %#v", joined)
	}

	// Now both parties (and the moderator) can message, including a tag that
	// should notify the tagged party.
	client.post("/disputes/messages", clientToken, map[string]any{"dispute_id": dispute.ID, "content": "here's what happened"}, http.StatusCreated, nil)
	client.post("/disputes/messages", adminToken, map[string]any{
		"dispute_id": dispute.ID, "content": "please share proof", "mentioned_user_id": musicianUser.ID,
	}, http.StatusCreated, nil)

	var disputeMessages []domain.ChatMessage
	client.get("/disputes/messages?dispute_id="+dispute.ID, musicianToken, http.StatusOK, &disputeMessages)
	if len(disputeMessages) < 4 { // opened + joined + client message + tag message
		t.Fatalf("expected at least 4 dispute messages, got %d: %#v", len(disputeMessages), disputeMessages)
	}

	// This contract has nothing funded at this point (it was already marked
	// complete above, before any milestone existed), so talent_amount must
	// be 0 — there's no escrow in scope to award anything from. Dispute
	// resolution's actual money-moving effect (partial settlement included)
	// is exercised below, at the milestone level.
	var resolved domain.Dispute
	client.post("/admin/disputes/resolve", adminToken, map[string]any{
		"dispute_id": dispute.ID, "resolution": "Resolved after review", "talent_amount": 0,
	}, http.StatusOK, &resolved)
	if resolved.Status != "resolved" {
		t.Fatalf("expected dispute status resolved, got %#v", resolved)
	}

	client.get("/wallet", clientToken, http.StatusOK, nil)
	client.get("/wallet/transactions", clientToken, http.StatusOK, nil)

	var proposed []domain.Milestone
	client.post("/milestones", clientToken, map[string]any{
		"contract_id": contracts[0].ID,
		"milestones":  []map[string]any{{"title": "Rehearsal complete", "amount": 100}},
	}, http.StatusCreated, &proposed)
	if len(proposed) != 1 || proposed[0].Status != "proposed" {
		t.Fatalf("expected one proposed milestone, got %+v", proposed)
	}

	var accepted domain.Milestone
	client.post("/milestones/accept", musicianToken, map[string]any{
		"contract_id": contracts[0].ID, "milestone_id": proposed[0].ID,
	}, http.StatusOK, &accepted)
	if accepted.Status != "accepted" {
		t.Fatalf("expected milestone accepted, got %s", accepted.Status)
	}

	// Funding a milestone now starts a PayPetal checkout too.
	var fundStart struct {
		PaymentURL string `json:"payment_url"`
		Reference  string `json:"reference"`
	}
	client.post("/milestones/fund", clientToken, map[string]any{
		"contract_id": contracts[0].ID, "milestone_id": proposed[0].ID,
	}, http.StatusOK, &fundStart)
	if fundStart.PaymentURL == "" || fundStart.Reference == "" {
		t.Fatalf("expected a payment URL and reference from milestone Fund, got %+v", fundStart)
	}
	app.paypetal.SimulatePaymentCompleted(fundStart.Reference)

	var funded domain.Milestone
	client.post("/milestones/fund/finalize", clientToken, map[string]any{"reference": fundStart.Reference}, http.StatusOK, &funded)
	if funded.Status != "funded" {
		t.Fatalf("expected milestone funded, got %s", funded.Status)
	}

	var released domain.Milestone
	client.post("/milestones/release", clientToken, map[string]any{
		"contract_id": contracts[0].ID, "milestone_id": proposed[0].ID,
	}, http.StatusOK, &released)
	if released.Status != "released" {
		t.Fatalf("expected milestone released, got %s", released.Status)
	}

	var musicianTxs []domain.Transaction
	client.get("/wallet/transactions", musicianToken, http.StatusOK, &musicianTxs)
	foundPayment := false
	for _, tx := range musicianTxs {
		// 90, not the milestone's agreed 100 — GigPurse's 10% commission
		// means only the talent's net take-home is ever actually escrowed
		// and released (see pricing.go).
		if tx.Type == "payment_received" && tx.Amount == 90 && tx.Reference == fundStart.Reference {
			foundPayment = true
		}
	}
	if !foundPayment {
		t.Fatalf("expected a payment_received transaction of 90 (100 minus 10%% commission) referencing the milestone's escrow agreement, got %#v", musicianTxs)
	}

	var musicianWallet domain.WalletSummary
	client.get("/wallet", musicianToken, http.StatusOK, &musicianWallet)
	// Just this milestone's net take-home — job hires no longer collect
	// anything upfront, so there's no job-level escrow to also show up here.
	if musicianWallet.TotalEarned != 90 {
		t.Fatalf("expected musician total_earned 90 (this milestone's take-home only), got %v", musicianWallet.TotalEarned)
	}

	// --- Cancellation & dispute-mediated partial settlement ---

	// A contract with no funded milestones ends cleanly, no dispute — the
	// direct-hire contract has none proposed on it at all.
	var endedCleanly domain.Contract
	client.post("/contracts/end", clientToken, map[string]any{"contract_id": acceptedDirectHire.ContractID}, http.StatusOK, &endedCleanly)
	if endedCleanly.Status != "cancelled" {
		t.Fatalf("expected contract with nothing funded to cancel cleanly, got status %q", endedCleanly.Status)
	}

	// Fund a second milestone on contracts[0], then have the talent pull it
	// back — this is the "funded, so it opens a dispute instead" path.
	var secondProposed []domain.Milestone
	client.post("/milestones", clientToken, map[string]any{
		"contract_id": contracts[0].ID,
		"milestones":  []map[string]any{{"title": "Second milestone", "amount": 200}},
	}, http.StatusCreated, &secondProposed)
	secondMilestoneID := secondProposed[0].ID
	client.post("/milestones/accept", musicianToken, map[string]any{"contract_id": contracts[0].ID, "milestone_id": secondMilestoneID}, http.StatusOK, nil)

	var secondFundStart struct {
		Reference string `json:"reference"`
	}
	client.post("/milestones/fund", clientToken, map[string]any{"contract_id": contracts[0].ID, "milestone_id": secondMilestoneID}, http.StatusOK, &secondFundStart)
	app.paypetal.SimulatePaymentCompleted(secondFundStart.Reference)
	client.post("/milestones/fund/finalize", clientToken, map[string]any{"reference": secondFundStart.Reference}, http.StatusOK, nil)

	var cancelled domain.Milestone
	client.post("/milestones/cancel", musicianToken, map[string]any{"contract_id": contracts[0].ID, "milestone_id": secondMilestoneID}, http.StatusOK, &cancelled)
	if cancelled.Status != "disputed" {
		t.Fatalf("expected pulling a funded milestone to leave it disputed, got %q", cancelled.Status)
	}

	var openDisputes []domain.Dispute
	client.get("/admin/disputes?status=open", adminToken, http.StatusOK, &openDisputes)
	var milestoneDispute domain.Dispute
	for _, d := range openDisputes {
		if d.MilestoneID == secondMilestoneID {
			milestoneDispute = d
		}
	}
	if milestoneDispute.ID == "" {
		t.Fatalf("expected an auto-opened dispute scoped to the pulled milestone, got %#v", openDisputes)
	}
	client.post("/disputes/join", adminToken, map[string]any{"dispute_id": milestoneDispute.ID}, http.StatusOK, nil)

	// Partial ruling: the escrowed take-home is 180 (200 minus 10% commission)
	// — award the talent 100 of it. The rest goes back to the client, and a
	// new settlement milestone should appear for exactly the awarded amount.
	var partialResolved domain.Dispute
	client.post("/admin/disputes/resolve", adminToken, map[string]any{
		"dispute_id": milestoneDispute.ID, "resolution": "Partial: some work was delivered", "talent_amount": 100,
	}, http.StatusOK, &partialResolved)
	if partialResolved.TalentAmountNaira != 100 {
		t.Fatalf("expected dispute to record a talent_amount of 100, got %v", partialResolved.TalentAmountNaira)
	}

	var afterResolve []domain.Milestone
	client.get("/milestones?contract_id="+contracts[0].ID, clientToken, http.StatusOK, &afterResolve)
	var original, settlement *domain.Milestone
	for i := range afterResolve {
		if afterResolve[i].ID == secondMilestoneID {
			original = &afterResolve[i]
		}
		if afterResolve[i].Title == "Dispute settlement" {
			settlement = &afterResolve[i]
		}
	}
	if original == nil || original.Status != "refunded" {
		t.Fatalf("expected the original disputed milestone refunded, got %#v", original)
	}
	if settlement == nil || settlement.Amount != 100 || settlement.Status != "accepted" {
		t.Fatalf("expected a new 'Dispute settlement' milestone for 100, accepted and awaiting funding, got %#v", settlement)
	}

	// The client funds the settlement milestone exactly like any other —
	// but it should auto-release the moment it's funded, no separate
	// Release click, since the moderator already ruled on it.
	var settlementFundStart struct {
		Reference string `json:"reference"`
	}
	client.post("/milestones/fund", clientToken, map[string]any{"contract_id": contracts[0].ID, "milestone_id": settlement.ID}, http.StatusOK, &settlementFundStart)
	app.paypetal.SimulatePaymentCompleted(settlementFundStart.Reference)
	var settlementFinalized domain.Milestone
	client.post("/milestones/fund/finalize", clientToken, map[string]any{"reference": settlementFundStart.Reference}, http.StatusOK, &settlementFinalized)
	if settlementFinalized.Status != "released" {
		t.Fatalf("expected the settlement milestone to auto-release on funding, got %q", settlementFinalized.Status)
	}

	var musicianTxsAfterSettlement []domain.Transaction
	client.get("/wallet/transactions", musicianToken, http.StatusOK, &musicianTxsAfterSettlement)
	foundSettlementPayment := false
	for _, tx := range musicianTxsAfterSettlement {
		// The full 100 — a dispute settlement is exempt from commission
		// (see milestoneUsecase.Fund), unlike the earlier 90-of-100 payment.
		if tx.Type == "payment_received" && tx.Amount == 100 && tx.Reference == settlementFundStart.Reference {
			foundSettlementPayment = true
		}
	}
	if !foundSettlementPayment {
		t.Fatalf("expected a commission-free payment_received of 100 for the settlement, got %#v", musicianTxsAfterSettlement)
	}

	// Full-release ruling: the moderator decides the talent did all the work
	// and should keep the entire escrowed amount, no refund at all. This
	// exercises milestoneUsecase.ReleaseDisputed directly (never covered by
	// the partial-resolution case above) — a live sandbox test caught this
	// path silently no-opping (releaseMilestone's CAS required status
	// "funded" to reach "released", but a disputed milestone is never
	// "funded" by resolution time, so the swap always lost and nothing
	// happened, with no error surfaced anywhere).
	var thirdProposed []domain.Milestone
	client.post("/milestones", clientToken, map[string]any{
		"contract_id": contracts[0].ID,
		"milestones":  []map[string]any{{"title": "Third milestone", "amount": 300}},
	}, http.StatusCreated, &thirdProposed)
	thirdMilestoneID := thirdProposed[0].ID
	client.post("/milestones/accept", musicianToken, map[string]any{"contract_id": contracts[0].ID, "milestone_id": thirdMilestoneID}, http.StatusOK, nil)

	var thirdFundStart struct {
		Reference string `json:"reference"`
	}
	client.post("/milestones/fund", clientToken, map[string]any{"contract_id": contracts[0].ID, "milestone_id": thirdMilestoneID}, http.StatusOK, &thirdFundStart)
	app.paypetal.SimulatePaymentCompleted(thirdFundStart.Reference)
	client.post("/milestones/fund/finalize", clientToken, map[string]any{"reference": thirdFundStart.Reference}, http.StatusOK, nil)

	client.post("/milestones/cancel", musicianToken, map[string]any{"contract_id": contracts[0].ID, "milestone_id": thirdMilestoneID}, http.StatusOK, nil)

	var openDisputesAfterThird []domain.Dispute
	client.get("/admin/disputes?status=open", adminToken, http.StatusOK, &openDisputesAfterThird)
	var thirdDispute domain.Dispute
	for _, d := range openDisputesAfterThird {
		if d.MilestoneID == thirdMilestoneID {
			thirdDispute = d
		}
	}
	if thirdDispute.ID == "" {
		t.Fatalf("expected an auto-opened dispute scoped to the third milestone, got %#v", openDisputesAfterThird)
	}
	client.post("/disputes/join", adminToken, map[string]any{"dispute_id": thirdDispute.ID}, http.StatusOK, nil)

	// Escrowed take-home is 270 (300 minus 10% commission) — award all of it.
	var fullReleaseResolved domain.Dispute
	client.post("/admin/disputes/resolve", adminToken, map[string]any{
		"dispute_id": thirdDispute.ID, "resolution": "Work was fully delivered", "talent_amount": 270,
	}, http.StatusOK, &fullReleaseResolved)

	var afterFullRelease []domain.Milestone
	client.get("/milestones?contract_id="+contracts[0].ID, clientToken, http.StatusOK, &afterFullRelease)
	var thirdAfter *domain.Milestone
	for i := range afterFullRelease {
		if afterFullRelease[i].ID == thirdMilestoneID {
			thirdAfter = &afterFullRelease[i]
		}
	}
	if thirdAfter == nil || thirdAfter.Status != "released" {
		t.Fatalf("expected a full-release ruling to actually release the disputed milestone, got %#v", thirdAfter)
	}

	var musicianTxsAfterFullRelease []domain.Transaction
	client.get("/wallet/transactions", musicianToken, http.StatusOK, &musicianTxsAfterFullRelease)
	foundFullReleasePayment := false
	for _, tx := range musicianTxsAfterFullRelease {
		if tx.Type == "payment_received" && tx.Amount == 270 && tx.Reference == thirdFundStart.Reference {
			foundFullReleasePayment = true
		}
	}
	if !foundFullReleasePayment {
		t.Fatalf("expected a payment_received of 270 for the full-release ruling, got %#v", musicianTxsAfterFullRelease)
	}

	client.get("/admin/analytics", adminToken, http.StatusOK, nil)
	client.get("/admin/users", adminToken, http.StatusOK, nil)
	client.get("/admin/jobs", adminToken, http.StatusOK, nil)
	client.delete("/admin/jobs", adminToken, map[string]any{"job_id": job.ID}, http.StatusOK, nil)

	if clientUser.ID == "" || musicianUser.ID == "" || adminUser.ID == "" {
		t.Fatal("expected seeded users to have IDs")
	}
}

// TestFinalizeFund_ConcurrentCallsAreSerialized reproduces, against the
// real HTTP server (not just the in-process usecase), the exact race
// found during live sandbox testing: PayPetal's webhook and GigPurse's own
// reconciler both call POST /milestones/fund/finalize for the same
// reference within moments of each other. Before the CompareAndSwapStatus
// guard, both calls slipped past the plain status check, double-recorded
// the funding transaction, and one silently stomped the other's status
// update. This fires many concurrent finalize calls and asserts exactly
// one funding transaction was recorded and the milestone ends up "funded"
// — not reverted, not duplicated.
func TestFinalizeFund_ConcurrentCallsAreSerialized(t *testing.T) {
	t.Setenv("JWT_SECRET", "api-simulation-secret")

	app := newTestApp()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	client := &apiClient{t: t, baseURL: server.URL, http: server.Client()}

	clientUser := client.signup("race-client@example.com", "password123", "client", "Race Client")
	musicianUser := client.signup("race-musician@example.com", "password123", "musician", "Race Musician")
	client.verifyEmail(app, clientUser.ID, "race-client@example.com", "111111")
	client.verifyEmail(app, musicianUser.ID, "race-musician@example.com", "222222")
	clientToken := client.login("race-client@example.com", "password123")
	musicianToken := client.login("race-musician@example.com", "password123")

	client.post("/payout-account/validate", musicianToken, map[string]any{
		"bank_code": "000", "account_number": "0000000000",
	}, http.StatusOK, nil)
	client.post("/payout-account", musicianToken, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0000000000",
	}, http.StatusOK, nil)
	client.post("/payout-account/validate", clientToken, map[string]any{
		"bank_code": "000", "account_number": "0000000000",
	}, http.StatusOK, nil)
	client.post("/payout-account", clientToken, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0000000000",
	}, http.StatusOK, nil)

	var job domain.Job
	client.post("/jobs", clientToken, map[string]any{
		"title": "Race condition test gig", "description": "Testing concurrent finalize", "instrument": "Guitar",
		"genre": "Test", "location": "Lagos", "budget": 5000, "experience_level": "intermediate",
		"duration": "less_than_1_week", "project_type": "one_time", "skills": []string{},
	}, http.StatusCreated, &job)
	client.post("/jobs/fund", clientToken, map[string]any{"job_id": job.ID}, http.StatusOK, nil)

	var application domain.JobApplication
	client.post("/jobs/apply", musicianToken, map[string]any{
		"job_id": job.ID, "proposal": "I'll do it", "price_bid": 5000,
	}, http.StatusCreated, &application)

	var contract domain.Contract
	client.post("/jobs/applications/accept", clientToken, map[string]any{"application_id": application.ID}, http.StatusOK, &contract)

	var proposed []domain.Milestone
	client.post("/milestones", clientToken, map[string]any{
		"contract_id": contract.ID,
		"milestones":  []map[string]any{{"title": "Race milestone", "amount": 5000}},
	}, http.StatusCreated, &proposed)
	milestoneID := proposed[0].ID
	client.post("/milestones/accept", musicianToken, map[string]any{"contract_id": contract.ID, "milestone_id": milestoneID}, http.StatusOK, nil)

	var fundStart struct {
		PaymentURL string `json:"payment_url"`
		Reference  string `json:"reference"`
	}
	client.post("/milestones/fund", clientToken, map[string]any{"contract_id": contract.ID, "milestone_id": milestoneID}, http.StatusOK, &fundStart)
	app.paypetal.SimulatePaymentCompleted(fundStart.Reference)

	const concurrentCalls = 20
	var wg sync.WaitGroup
	statuses := make([]int, concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body, _ := json.Marshal(map[string]any{"reference": fundStart.Reference})
			req, err := http.NewRequest(http.MethodPost, server.URL+"/milestones/fund/finalize", bytes.NewReader(body))
			if err != nil {
				t.Errorf("build request %d: %v", i, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+clientToken)
			resp, err := server.Client().Do(req)
			if err != nil {
				t.Errorf("request %d failed: %v", i, err)
				return
			}
			defer resp.Body.Close()
			statuses[i] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	for i, code := range statuses {
		if code != http.StatusOK {
			t.Errorf("concurrent finalize call %d: expected 200, got %d", i, code)
		}
	}

	var finalMilestones []domain.Milestone
	client.get("/milestones?contract_id="+contract.ID, clientToken, http.StatusOK, &finalMilestones)
	var final *domain.Milestone
	for i := range finalMilestones {
		if finalMilestones[i].ID == milestoneID {
			final = &finalMilestones[i]
		}
	}
	if final == nil {
		t.Fatal("milestone vanished")
	}
	if final.Status != "funded" {
		t.Fatalf("expected milestone to end up \"funded\" after %d concurrent finalize calls, got %q", concurrentCalls, final.Status)
	}

	var clientTxs []domain.Transaction
	client.get("/wallet/transactions", clientToken, http.StatusOK, &clientTxs)
	holdCount := 0
	for _, tx := range clientTxs {
		if tx.Type == "escrow_hold" && tx.Reference == fundStart.Reference {
			holdCount++
		}
	}
	if holdCount != 1 {
		t.Fatalf("expected exactly 1 escrow_hold transaction after %d concurrent finalize calls, got %d", concurrentCalls, holdCount)
	}
}

type testApp struct {
	mux             *http.ServeMux
	resetRepo       *memoryPasswordResetRepo
	emailVerifyRepo *memoryEmailVerificationRepo
	paypetal        *memory.PayPetalFake
}

func newTestApp() *testApp {
	userRepo := newMemoryUserRepo()
	jobRepo := newMemoryJobRepo()
	chatRepo := newMemoryChatRepo()
	contractRepo := newMemoryContractRepo()
	reviewRepo := newMemoryReviewRepo()
	notifRepo := newMemoryNotificationRepo()
	resetRepo := newMemoryPasswordResetRepo()
	emailVerifyRepo := newMemoryEmailVerificationRepo()
	disputeRepo := newMemoryDisputeRepo()
	walletRepo := memory.NewWalletRepository()
	milestoneRepo := memory.NewMilestoneRepository()
	escrowRepo := memory.NewEscrowAgreementRepository()
	paypetalFake := memory.NewPayPetalFake()
	hub := delivery.NewHub()
	const testFrontendBaseURL = "https://gigpurse.test"

	userUsecase := usecase.NewUserUsecaseWithVerification(userRepo, resetRepo, emailVerifyRepo, hub)
	jobUsecase := usecase.NewJobUsecase(jobRepo, userRepo, contractRepo, notifRepo, walletRepo, reviewRepo, paypetalFake, escrowRepo, milestoneRepo, testFrontendBaseURL)
	chatUsecase := usecase.NewChatUsecase(chatRepo, userRepo, notifRepo)
	contractUsecase := usecase.NewContractUsecase(contractRepo, jobRepo, notifRepo, userRepo, walletRepo, paypetalFake, escrowRepo)
	reviewUsecase := usecase.NewReviewUsecase(reviewRepo, contractRepo, notifRepo)
	notifUsecase := usecase.NewNotificationUsecase(notifRepo)
	dashboardUsecase := usecase.NewDashboardUsecase(jobUsecase, contractUsecase, reviewUsecase)
	adminUsecase := &memoryAdminUsecase{users: userRepo, jobs: jobRepo, chats: chatRepo, contracts: contractRepo, disputes: disputeRepo, milestones: milestoneRepo, escrow: escrowRepo, jobUsecase: jobUsecase}
	milestoneUsecase := usecase.NewMilestoneUsecase(milestoneRepo, contractRepo, walletRepo, notifRepo, chatRepo, hub, paypetalFake, userRepo, escrowRepo, testFrontendBaseURL)
	walletUsecase := usecase.NewWalletUsecase(walletRepo, userRepo, escrowRepo, jobRepo, milestoneRepo, jobUsecase, milestoneUsecase)
	disputeUsecase := usecase.NewDisputeUsecase(disputeRepo, contractRepo, notifRepo, chatRepo, userRepo, jobRepo, walletRepo, milestoneUsecase, paypetalFake, escrowRepo)
	payoutAccountUsecase := usecase.NewPayoutAccountUsecase(paypetalFake, userRepo, contractRepo, milestoneRepo, notifRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"test-online"}`))
	})
	delivery.NewUserHandler(userUsecase, contractRepo).RegisterRoutes(mux)
	delivery.NewJobHandler(jobUsecase).RegisterRoutes(mux)
	delivery.NewChatHandler(chatUsecase, disputeUsecase, hub).RegisterRoutes(mux)
	delivery.NewContractHandler(contractUsecase, disputeUsecase).RegisterRoutes(mux)
	delivery.NewReviewHandler(reviewUsecase).RegisterRoutes(mux)
	delivery.NewNotificationHandler(notifUsecase).RegisterRoutes(mux)
	delivery.NewDisputeHandler(disputeUsecase, hub).RegisterRoutes(mux)
	delivery.NewDashboardHandler(dashboardUsecase).RegisterRoutes(mux)
	delivery.NewAdminHandler(adminUsecase).RegisterRoutes(mux)
	delivery.NewWalletHandler(walletUsecase).RegisterRoutes(mux)
	delivery.NewMilestoneHandler(milestoneUsecase, disputeUsecase).RegisterRoutes(mux)
	delivery.NewPayoutAccountHandler(payoutAccountUsecase).RegisterRoutes(mux)

	return &testApp{mux: mux, resetRepo: resetRepo, emailVerifyRepo: emailVerifyRepo, paypetal: paypetalFake}
}

type apiClient struct {
	t       *testing.T
	baseURL string
	http    *http.Client
}

func (c *apiClient) signup(email, password, role, name string) domain.User {
	var user domain.User
	c.post("/auth/signup", "", map[string]any{
		"email": email, "password": password, "role": role, "name": name,
		"phone": "+234801" + email[:7], "accepted_terms": true,
	}, http.StatusCreated, &user)
	return user
}

func (c *apiClient) login(email, password string) string {
	var res struct {
		Token string      `json:"token"`
		User  domain.User `json:"user"`
	}
	c.post("/auth/login", "", map[string]any{"email": email, "password": password}, http.StatusOK, &res)
	if res.Token == "" {
		c.t.Fatal("expected login token")
	}
	return res.Token
}

func (c *apiClient) verifyEmail(app *testApp, userID, email, code string) {
	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email)) + ":" + strings.TrimSpace(code)))
	err := app.emailVerifyRepo.Create(context.Background(), &domain.EmailVerificationToken{
		UserID:    userID,
		TokenHash: hex.EncodeToString(hash[:]),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	})
	if err != nil {
		c.t.Fatalf("seed email verification code: %v", err)
	}
	c.post("/auth/email-verification/confirm", "", map[string]any{"email": email, "code": code}, http.StatusOK, nil)
}

func (c *apiClient) get(path, token string, want int, out any) {
	c.request(http.MethodGet, path, token, nil, want, out)
}

func (c *apiClient) post(path, token string, body any, want int, out any) {
	c.request(http.MethodPost, path, token, body, want, out)
}

func (c *apiClient) put(path, token string, body any, want int, out any) {
	c.request(http.MethodPut, path, token, body, want, out)
}

func (c *apiClient) delete(path, token string, body any, want int, out any) {
	c.request(http.MethodDelete, path, token, body, want, out)
}

func (c *apiClient) request(method, path, token string, body any, want int, out any) {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal request %s %s: %v", method, path, err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		c.t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		c.t.Fatalf("%s %s status=%d want=%d", method, path, resp.StatusCode, want)
	}
	if out != nil {
		var envelope struct {
			Success bool            `json:"success"`
			Data    json.RawMessage `json:"data"`
			Error   any             `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			c.t.Fatalf("decode %s %s: %v", method, path, err)
		}
		if !envelope.Success {
			c.t.Fatalf("%s %s returned unsuccessful envelope: %#v", method, path, envelope.Error)
		}
		if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
			return
		}
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			c.t.Fatalf("decode data %s %s: %v", method, path, err)
		}
	}
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	return conn
}

type memoryUserRepo struct {
	mu     sync.RWMutex
	next   int
	users  map[string]*domain.User
	emails map[string]string
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{users: map[string]*domain.User{}, emails: map[string]string{}}
}

func (r *memoryUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.emails[user.Email]; exists {
		return errors.New("email already registered")
	}
	r.next++
	user.ID = fmt.Sprintf("usr_%d", r.next)
	cp := *user
	r.users[user.ID] = &cp
	r.emails[user.Email] = user.ID
	return nil
}

func (r *memoryUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *user
	return &cp, nil
}

func (r *memoryUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.emails[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *r.users[id]
	return &cp, nil
}

func (r *memoryUserRepo) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *user
	r.users[user.ID] = &cp
	r.emails[user.Email] = user.ID
	return nil
}

func (r *memoryUserRepo) ListMusicians(ctx context.Context, filter domain.MusicianFilter) ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var users []*domain.User
	for _, user := range r.users {
		if user.Role != "musician" {
			continue
		}
		if filter.Location != "" && !strings.Contains(strings.ToLower(user.Location), strings.ToLower(filter.Location)) {
			continue
		}
		if user.MusicianProfile != nil {
			if filter.Genre != "" && !containsSubstringFold(user.MusicianProfile.Genres, filter.Genre) {
				continue
			}
			if filter.Instrument != "" && !containsSubstringFold(user.MusicianProfile.Instruments, filter.Instrument) {
				continue
			}
			if filter.MinExp > 0 && user.MusicianProfile.ExperienceYears < filter.MinExp {
				continue
			}
		}
		cp := *user
		users = append(users, &cp)
	}
	return users, nil
}

func containsSubstringFold(values []string, target string) bool {
	for _, v := range values {
		if strings.Contains(strings.ToLower(v), strings.ToLower(target)) {
			return true
		}
	}
	return false
}

func (r *memoryUserRepo) listAll() []*domain.User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]*domain.User, 0, len(r.users))
	for _, user := range r.users {
		cp := *user
		users = append(users, &cp)
	}
	return users
}

type memoryJobRepo struct {
	mu   sync.RWMutex
	next int
	jobs map[string]*domain.Job
	apps map[string]*domain.JobApplication
}

func newMemoryJobRepo() *memoryJobRepo {
	return &memoryJobRepo{jobs: map[string]*domain.Job{}, apps: map[string]*domain.JobApplication{}}
}

func (r *memoryJobRepo) Create(ctx context.Context, job *domain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	job.ID = fmt.Sprintf("job_%d", r.next)
	cp := *job
	r.jobs[job.ID] = &cp
	return nil
}

func (r *memoryJobRepo) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, ok := r.jobs[id]
	if !ok {
		return nil, errors.New("job not found")
	}
	cp := *job
	return &cp, nil
}

func (r *memoryJobRepo) Update(ctx context.Context, job *domain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *job
	r.jobs[job.ID] = &cp
	return nil
}

func (r *memoryJobRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobs, id)
	return nil
}

func (r *memoryJobRepo) List(ctx context.Context, filter domain.JobFilter) ([]*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var jobs []*domain.Job
	for _, job := range r.jobs {
		if filter.Status != "" && job.Status != filter.Status {
			continue
		}
		if filter.Genre != "" && !strings.Contains(strings.ToLower(job.Genre), strings.ToLower(filter.Genre)) {
			continue
		}
		if filter.Instrument != "" && !strings.Contains(strings.ToLower(job.Instrument), strings.ToLower(filter.Instrument)) {
			continue
		}
		if filter.Location != "" && !strings.Contains(strings.ToLower(job.Location), strings.ToLower(filter.Location)) {
			continue
		}
		if filter.ClientID != "" && job.ClientID != filter.ClientID {
			continue
		}
		if filter.MinBudget > 0 && job.Budget < filter.MinBudget {
			continue
		}
		if filter.MaxBudget > 0 && job.Budget > filter.MaxBudget {
			continue
		}
		cp := *job
		jobs = append(jobs, &cp)
	}
	return jobs, nil
}

func (r *memoryJobRepo) CreateApplication(ctx context.Context, app *domain.JobApplication) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	app.ID = fmt.Sprintf("app_%d", r.next)
	cp := *app
	r.apps[app.ID] = &cp
	return nil
}

func (r *memoryJobRepo) GetApplicationByID(ctx context.Context, id string) (*domain.JobApplication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[id]
	if !ok {
		return nil, errors.New("application not found")
	}
	cp := *app
	return &cp, nil
}

func (r *memoryJobRepo) UpdateApplication(ctx context.Context, app *domain.JobApplication) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *app
	r.apps[app.ID] = &cp
	return nil
}

func (r *memoryJobRepo) ListApplications(ctx context.Context, jobID string) ([]*domain.JobApplication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var apps []*domain.JobApplication
	for _, app := range r.apps {
		if app.JobID == jobID {
			cp := *app
			apps = append(apps, &cp)
		}
	}
	return apps, nil
}

func (r *memoryJobRepo) CountApplications(ctx context.Context, jobID string) (int, error) {
	apps, err := r.ListApplications(ctx, jobID)
	return len(apps), err
}

func (r *memoryJobRepo) ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*domain.JobApplication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var apps []*domain.JobApplication
	for _, app := range r.apps {
		if app.MusicianID == musicianID {
			cp := *app
			apps = append(apps, &cp)
		}
	}
	return apps, nil
}

func (r *memoryJobRepo) delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobs, id)
}

type memoryChatRepo struct {
	mu       sync.RWMutex
	next     int
	messages []*domain.ChatMessage
}

func newMemoryChatRepo() *memoryChatRepo { return &memoryChatRepo{} }

func (r *memoryChatRepo) SaveMessage(ctx context.Context, msg *domain.ChatMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	msg.ID = fmt.Sprintf("msg_%d", r.next)
	cp := *msg
	r.messages = append(r.messages, &cp)
	return nil
}

func (r *memoryChatRepo) GetChatHistory(ctx context.Context, user1, user2 string) ([]*domain.ChatMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*domain.ChatMessage
	for _, msg := range r.messages {
		if (msg.SenderID == user1 && msg.RecvID == user2) || (msg.SenderID == user2 && msg.RecvID == user1) {
			cp := *msg
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *memoryChatRepo) GetRecentChats(ctx context.Context, userID string) ([]*domain.ChatMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := map[string]*domain.ChatMessage{}
	for _, msg := range r.messages {
		partner := msg.SenderID
		if msg.SenderID == userID {
			partner = msg.RecvID
		} else if msg.RecvID != userID {
			continue
		}
		if current, ok := latest[partner]; !ok || msg.Timestamp.After(current.Timestamp) {
			cp := *msg
			latest[partner] = &cp
		}
	}
	var out []*domain.ChatMessage
	for _, msg := range latest {
		out = append(out, msg)
	}
	return out, nil
}

func (r *memoryChatRepo) ListByDispute(ctx context.Context, disputeID string) ([]*domain.ChatMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*domain.ChatMessage
	for _, msg := range r.messages {
		if msg.DisputeID == disputeID {
			cp := *msg
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *memoryChatRepo) MarkConversationRead(ctx context.Context, recvID, senderID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, msg := range r.messages {
		if msg.SenderID == senderID && msg.RecvID == recvID {
			msg.Read = true
		}
	}
	return nil
}

func (r *memoryChatRepo) ListUnreadOlderThan(ctx context.Context, cutoff time.Time) ([]*domain.ChatMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ChatMessage{}
	for _, msg := range r.messages {
		if !msg.Read && msg.ReminderEmailSentAt == nil && msg.RecvID != "" && msg.Timestamp.Before(cutoff) {
			cp := *msg
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *memoryChatRepo) MarkReminderEmailSent(ctx context.Context, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	now := time.Now()
	for _, msg := range r.messages {
		if idSet[msg.ID] {
			msg.ReminderEmailSentAt = &now
		}
	}
	return nil
}

func (r *memoryChatRepo) count() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.messages))
}

type memoryContractRepo struct {
	mu          sync.RWMutex
	next        int
	contracts   map[string]*domain.Contract
	directHires map[string]*domain.DirectHireRequest
}

func newMemoryContractRepo() *memoryContractRepo {
	return &memoryContractRepo{contracts: map[string]*domain.Contract{}, directHires: map[string]*domain.DirectHireRequest{}}
}

func (r *memoryContractRepo) Create(ctx context.Context, contract *domain.Contract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	contract.ID = fmt.Sprintf("con_%d", r.next)
	cp := *contract
	r.contracts[contract.ID] = &cp
	return nil
}

func (r *memoryContractRepo) GetByID(ctx context.Context, id string) (*domain.Contract, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	contract, ok := r.contracts[id]
	if !ok {
		return nil, errors.New("contract not found")
	}
	cp := *contract
	return &cp, nil
}

func (r *memoryContractRepo) GetByJobID(ctx context.Context, jobID string) (*domain.Contract, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, contract := range r.contracts {
		if contract.JobID == jobID {
			cp := *contract
			return &cp, nil
		}
	}
	return nil, errors.New("contract not found")
}

func (r *memoryContractRepo) Update(ctx context.Context, contract *domain.Contract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *contract
	r.contracts[contract.ID] = &cp
	return nil
}

func (r *memoryContractRepo) ListForUser(ctx context.Context, userID, role string) ([]*domain.Contract, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var contracts []*domain.Contract
	for _, contract := range r.contracts {
		if contract.ClientID == userID || contract.MusicianID == userID || role == "admin" {
			cp := *contract
			contracts = append(contracts, &cp)
		}
	}
	return contracts, nil
}

func (r *memoryContractRepo) CreateDirectHireRequest(ctx context.Context, req *domain.DirectHireRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	req.ID = fmt.Sprintf("dh_%d", r.next)
	cp := *req
	r.directHires[req.ID] = &cp
	return nil
}

func (r *memoryContractRepo) GetDirectHireRequestByID(ctx context.Context, id string) (*domain.DirectHireRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	req, ok := r.directHires[id]
	if !ok {
		return nil, errors.New("direct hire request not found")
	}
	cp := *req
	return &cp, nil
}

func (r *memoryContractRepo) UpdateDirectHireRequest(ctx context.Context, req *domain.DirectHireRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *req
	r.directHires[req.ID] = &cp
	return nil
}

func (r *memoryContractRepo) ListDirectHireRequestsForUser(ctx context.Context, userID, role, status string) ([]*domain.DirectHireRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var requests []*domain.DirectHireRequest
	for _, req := range r.directHires {
		if status != "" && req.Status != status {
			continue
		}
		if req.ClientID == userID || req.MusicianID == userID || role == "admin" {
			cp := *req
			requests = append(requests, &cp)
		}
	}
	return requests, nil
}

func (r *memoryContractRepo) count() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.contracts))
}

type memoryReviewRepo struct {
	mu      sync.RWMutex
	next    int
	reviews map[string]*domain.Review
}

func newMemoryReviewRepo() *memoryReviewRepo {
	return &memoryReviewRepo{reviews: map[string]*domain.Review{}}
}

func (r *memoryReviewRepo) Create(ctx context.Context, review *domain.Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	review.ID = fmt.Sprintf("rev_%d", r.next)
	cp := *review
	r.reviews[review.ID] = &cp
	return nil
}

func (r *memoryReviewRepo) ListByReviewee(ctx context.Context, revieweeID string) ([]*domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var reviews []*domain.Review
	for _, review := range r.reviews {
		if review.RevieweeID == revieweeID {
			cp := *review
			reviews = append(reviews, &cp)
		}
	}
	return reviews, nil
}

func (r *memoryReviewRepo) GetByContractAndReviewer(ctx context.Context, contractID, reviewerID string) (*domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, review := range r.reviews {
		if review.ContractID == contractID && review.ReviewerID == reviewerID {
			cp := *review
			return &cp, nil
		}
	}
	return nil, errors.New("review not found")
}

type memoryNotificationRepo struct {
	mu     sync.RWMutex
	next   int
	notifs map[string]*domain.Notification
}

func newMemoryNotificationRepo() *memoryNotificationRepo {
	return &memoryNotificationRepo{notifs: map[string]*domain.Notification{}}
}

func (r *memoryNotificationRepo) Create(ctx context.Context, notif *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	notif.ID = fmt.Sprintf("not_%d", r.next)
	cp := *notif
	r.notifs[notif.ID] = &cp
	return nil
}

func (r *memoryNotificationRepo) ListForUser(ctx context.Context, userID string) ([]*domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var notifs []*domain.Notification
	for _, notif := range r.notifs {
		if notif.UserID == userID {
			cp := *notif
			notifs = append(notifs, &cp)
		}
	}
	sort.Slice(notifs, func(i, j int) bool { return notifs[i].CreatedAt.After(notifs[j].CreatedAt) })
	return notifs, nil
}

func (r *memoryNotificationRepo) MarkAsRead(ctx context.Context, notifID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	notif, ok := r.notifs[notifID]
	if !ok || notif.UserID != userID {
		return errors.New("notification not found")
	}
	notif.IsRead = true
	return nil
}

type memoryPasswordResetRepo struct {
	mu     sync.RWMutex
	tokens map[string]*domain.PasswordResetToken
}

func newMemoryPasswordResetRepo() *memoryPasswordResetRepo {
	return &memoryPasswordResetRepo{tokens: map[string]*domain.PasswordResetToken{}}
}

func (r *memoryPasswordResetRepo) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	token.ID = fmt.Sprintf("rst_%d", len(r.tokens)+1)
	cp := *token
	r.tokens[token.TokenHash] = &cp
	return nil
}

func (r *memoryPasswordResetRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	token, ok := r.tokens[tokenHash]
	if !ok {
		return nil, errors.New("password reset token not found")
	}
	cp := *token
	return &cp, nil
}

func (r *memoryPasswordResetRepo) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, token := range r.tokens {
		if token.ID == id {
			token.UsedAt = usedAt
			return nil
		}
	}
	return errors.New("password reset token not found")
}

type memoryEmailVerificationRepo struct {
	mu     sync.RWMutex
	tokens map[string]*domain.EmailVerificationToken
}

func newMemoryEmailVerificationRepo() *memoryEmailVerificationRepo {
	return &memoryEmailVerificationRepo{tokens: map[string]*domain.EmailVerificationToken{}}
}

func (r *memoryEmailVerificationRepo) Create(ctx context.Context, token *domain.EmailVerificationToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	token.ID = fmt.Sprintf("emv_%d", len(r.tokens)+1)
	cp := *token
	r.tokens[token.TokenHash] = &cp
	return nil
}

func (r *memoryEmailVerificationRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	token, ok := r.tokens[tokenHash]
	if !ok {
		return nil, errors.New("email verification token not found")
	}
	cp := *token
	return &cp, nil
}

func (r *memoryEmailVerificationRepo) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, token := range r.tokens {
		if token.ID == id {
			token.UsedAt = usedAt
			return nil
		}
	}
	return errors.New("email verification token not found")
}

type memoryDisputeRepo struct {
	mu       sync.RWMutex
	next     int
	disputes map[string]*domain.Dispute
}

func newMemoryDisputeRepo() *memoryDisputeRepo {
	return &memoryDisputeRepo{disputes: map[string]*domain.Dispute{}}
}

func (r *memoryDisputeRepo) Create(ctx context.Context, dispute *domain.Dispute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	dispute.ID = fmt.Sprintf("dsp_%d", r.next)
	cp := *dispute
	r.disputes[dispute.ID] = &cp
	return nil
}

func (r *memoryDisputeRepo) GetByID(ctx context.Context, id string) (*domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dispute, ok := r.disputes[id]
	if !ok {
		return nil, errors.New("dispute not found")
	}
	cp := *dispute
	return &cp, nil
}

func (r *memoryDisputeRepo) Update(ctx context.Context, dispute *domain.Dispute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *dispute
	r.disputes[dispute.ID] = &cp
	return nil
}

func (r *memoryDisputeRepo) List(ctx context.Context, status string) ([]*domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var disputes []*domain.Dispute
	for _, dispute := range r.disputes {
		if status == "" || dispute.Status == status {
			cp := *dispute
			disputes = append(disputes, &cp)
		}
	}
	return disputes, nil
}

func (r *memoryDisputeRepo) ListForUser(ctx context.Context, userID string) ([]*domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var disputes []*domain.Dispute
	for _, dispute := range r.disputes {
		if dispute.ClientID == userID || dispute.MusicianID == userID || dispute.OpenedByID == userID {
			cp := *dispute
			disputes = append(disputes, &cp)
		}
	}
	return disputes, nil
}

func (r *memoryDisputeRepo) count() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.disputes))
}

type memoryAdminUsecase struct {
	users      *memoryUserRepo
	jobs       *memoryJobRepo
	chats      *memoryChatRepo
	contracts  *memoryContractRepo
	disputes   *memoryDisputeRepo
	milestones domain.MilestoneRepository
	escrow     domain.EscrowAgreementRepository
	jobUsecase domain.JobUsecase
}

func (u *memoryAdminUsecase) GetAnalytics(ctx context.Context) (*domain.AdminAnalytics, error) {
	return &domain.AdminAnalytics{
		TotalUsers:     int64(len(u.users.listAll())),
		TotalJobs:      int64(len(u.jobs.jobs)),
		TotalMessages:  u.chats.count(),
		TotalContracts: u.contracts.count(),
		TotalDisputes:  u.disputes.count(),
	}, nil
}

func (u *memoryAdminUsecase) ListAllUsers(ctx context.Context) ([]*domain.User, error) {
	return u.users.listAll(), nil
}

func (u *memoryAdminUsecase) ListAllJobs(ctx context.Context) ([]*domain.Job, error) {
	return u.jobs.List(ctx, domain.JobFilter{})
}

func (u *memoryAdminUsecase) GetJobDetail(ctx context.Context, jobID string) (*domain.AdminJobDetail, error) {
	job, err := u.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	apps, err := u.jobUsecase.ListJobApplications(ctx, jobID)
	if err != nil {
		return nil, err
	}
	detail := &domain.AdminJobDetail{Job: job, Applications: apps}

	contract, err := u.contracts.GetByJobID(ctx, jobID)
	if err != nil || contract == nil {
		return detail, nil
	}
	detail.Contract = contract

	milestones, err := u.milestones.ListByContract(ctx, contract.ID)
	if err != nil {
		return detail, nil
	}
	for _, m := range milestones {
		summary := &domain.AdminMilestoneSummary{
			ID: m.ID, Title: m.Title, Amount: m.Amount, Status: m.Status,
			DueDate: m.DueDate, CreatedAt: m.CreatedAt,
		}
		if m.EscrowReference != "" {
			if agreement, err := u.escrow.GetByReference(ctx, m.EscrowReference); err == nil {
				summary.EscrowStatus = agreement.Status
				summary.PayoutStatus = agreement.PayoutStatus
				summary.RefundStatus = agreement.RefundStatus
				summary.PlatformFee = agreement.PlatformFeeNaira
			}
		}
		detail.Milestones = append(detail.Milestones, summary)
	}
	return detail, nil
}

func (u *memoryAdminUsecase) DeleteJobListing(ctx context.Context, jobID string) error {
	u.jobs.delete(jobID)
	return nil
}

func (u *memoryAdminUsecase) InviteModerator(ctx context.Context, email, name string) (*domain.User, error) {
	if existing, err := u.users.GetByEmail(ctx, email); err == nil && existing != nil {
		return nil, fmt.Errorf("a user with this email already exists (role: %s)", existing.Role)
	}
	user := &domain.User{Email: email, Role: "moderator", Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := u.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *memoryAdminUsecase) ListModerators(ctx context.Context) ([]*domain.User, error) {
	var out []*domain.User
	for _, user := range u.users.listAll() {
		if user.Role == "moderator" || user.Role == "revoked_moderator" {
			out = append(out, user)
		}
	}
	return out, nil
}

func (u *memoryAdminUsecase) SetModeratorStatus(ctx context.Context, userID string, active bool) error {
	user, err := u.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if active {
		user.Role = "moderator"
	} else {
		user.Role = "revoked_moderator"
	}
	return u.users.Update(ctx, user)
}

// engagementFake is a low-fidelity stand-in for admin_usecase.go's real
// per-user Mongo aggregations — good enough to satisfy domain.AdminUsecase
// and return plausible, non-crashing data. Nothing in this suite currently
// asserts on specific engagement numbers, so exact parity with the real
// aggregation logic isn't required here.
func (u *memoryAdminUsecase) engagementFor(ctx context.Context, user *domain.User, windowDays int, role string) *domain.EngagementSummary {
	since := time.Now().AddDate(0, 0, -windowDays)
	var last time.Time
	var windowCount, allTimeCount int64
	bump := func(t time.Time) {
		if t.After(last) {
			last = t
		}
		allTimeCount++
		if !t.Before(since) {
			windowCount++
		}
	}

	if role == "musician" {
		apps, _ := u.jobs.ListApplicationsByMusician(ctx, user.ID)
		for _, a := range apps {
			bump(a.CreatedAt)
		}
	}
	contracts, _ := u.contracts.ListForUser(ctx, user.ID, role)
	var gigs int64
	for _, c := range contracts {
		bump(c.CreatedAt)
		if c.Status == "completed" {
			gigs++
		}
	}
	hires, _ := u.contracts.ListDirectHireRequestsForUser(ctx, user.ID, role, "")
	for _, h := range hires {
		bump(h.CreatedAt)
	}
	for _, m := range u.chats.messages {
		if m.SenderID == user.ID {
			bump(m.Timestamp)
		}
	}

	var financial float64
	if role == "client" {
		agreements, _ := u.escrow.ListByInitiator(ctx, user.ID)
		for _, a := range agreements {
			if a.Status != "PENDING" {
				financial += a.AmountNaira + a.PlatformFeeNaira
			}
		}
	}

	summary := &domain.EngagementSummary{
		UserID: user.ID, Name: user.Name, Email: user.Email, JoinedAt: user.CreatedAt,
		EngagementCount: windowCount, GigsCount: gigs, FinancialTotal: financial,
	}
	if !last.IsZero() {
		summary.LastEngagedAt = &last
	}
	if role == "client" {
		months := time.Since(user.CreatedAt).Hours() / 24 / 30
		if months < 1 {
			months = 1
		}
		summary.AvgEngagementPerMonth = float64(allTimeCount) / months
	}
	return summary
}

func (u *memoryAdminUsecase) ListTalentEngagement(ctx context.Context, windowDays int) ([]*domain.EngagementSummary, error) {
	var out []*domain.EngagementSummary
	for _, user := range u.users.listAll() {
		if user.Role == "musician" {
			out = append(out, u.engagementFor(ctx, user, windowDays, "musician"))
		}
	}
	return out, nil
}

func (u *memoryAdminUsecase) ListClientEngagement(ctx context.Context, windowDays int) ([]*domain.EngagementSummary, error) {
	var out []*domain.EngagementSummary
	for _, user := range u.users.listAll() {
		if user.Role == "client" {
			out = append(out, u.engagementFor(ctx, user, windowDays, "client"))
		}
	}
	return out, nil
}

// TestMemoryChatRepo_MarkConversationRead verifies the real repository
// (not a test stub) used by the running app: marking a conversation read
// only flips messages actually sent by that specific sender to that
// specific recipient, leaving everything else untouched.
func TestMemoryChatRepo_MarkConversationRead(t *testing.T) {
	repo := newMemoryChatRepo()
	ctx := context.Background()

	msgs := []*domain.ChatMessage{
		{SenderID: "a", RecvID: "b", Content: "hi from a", Timestamp: time.Now()},
		{SenderID: "a", RecvID: "b", Content: "hi again", Timestamp: time.Now()},
		{SenderID: "b", RecvID: "a", Content: "reply from b", Timestamp: time.Now()},
		{SenderID: "c", RecvID: "b", Content: "unrelated sender", Timestamp: time.Now()},
	}
	for _, m := range msgs {
		if err := repo.SaveMessage(ctx, m); err != nil {
			t.Fatalf("save message: %v", err)
		}
	}

	if err := repo.MarkConversationRead(ctx, "b", "a"); err != nil {
		t.Fatalf("mark conversation read: %v", err)
	}

	history, err := repo.GetChatHistory(ctx, "a", "b")
	if err != nil {
		t.Fatalf("get chat history: %v", err)
	}
	for _, m := range history {
		if m.SenderID == "a" && m.RecvID == "b" && !m.Read {
			t.Fatalf("expected a->b message to be marked read: %#v", m)
		}
		if m.SenderID == "b" && m.RecvID == "a" && m.Read {
			t.Fatalf("expected b->a message to remain unread (b hasn't read a's messages, only the reverse): %#v", m)
		}
	}

	fromC, err := repo.GetChatHistory(ctx, "b", "c")
	if err != nil {
		t.Fatalf("get chat history: %v", err)
	}
	for _, m := range fromC {
		if m.Read {
			t.Fatalf("expected the unrelated c->b message to remain unread: %#v", m)
		}
	}
}

// TestMemoryChatRepo_UnreadDigestFlow exercises ListUnreadOlderThan and
// MarkReminderEmailSent together: a stale unread message should surface
// once, and stop surfacing once reminded — the exact idempotency
// StartUnreadEmailScanner depends on to avoid re-emailing every tick.
func TestMemoryChatRepo_UnreadDigestFlow(t *testing.T) {
	repo := newMemoryChatRepo()
	ctx := context.Background()

	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now()

	stale := &domain.ChatMessage{SenderID: "a", RecvID: "b", Content: "old and unread", Timestamp: old}
	fresh := &domain.ChatMessage{SenderID: "a", RecvID: "b", Content: "just sent", Timestamp: recent}
	if err := repo.SaveMessage(ctx, stale); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := repo.SaveMessage(ctx, fresh); err != nil {
		t.Fatalf("save: %v", err)
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	unread, err := repo.ListUnreadOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("list unread: %v", err)
	}
	if len(unread) != 1 || unread[0].ID != stale.ID {
		t.Fatalf("expected only the stale message, got %#v", unread)
	}

	if err := repo.MarkReminderEmailSent(ctx, []string{stale.ID}); err != nil {
		t.Fatalf("mark reminder sent: %v", err)
	}

	unreadAgain, err := repo.ListUnreadOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("list unread again: %v", err)
	}
	if len(unreadAgain) != 0 {
		t.Fatalf("expected the reminded message to stop appearing, got %#v", unreadAgain)
	}
}

// TestPayoutAccount_RelinkSelfHeals covers the real production bug: PayPetal
// allows exactly one bank account per customer, ever, and a second
// LinkPayoutAccount call for the same customer fails with "A bank account
// has already been added for this customer" — confirmed live against
// PayPetal's sandbox. Before the fix this left a user permanently stuck,
// unable to add or change a payout account after the first attempt.
func TestPayoutAccount_RelinkSelfHeals(t *testing.T) {
	t.Setenv("JWT_SECRET", "payout-relink-secret")

	app := newTestApp()
	server := httptest.NewServer(app.mux)
	defer server.Close()

	client := &apiClient{t: t, baseURL: server.URL, http: server.Client()}

	musicianUser := client.signup("relink-musician@example.com", "password123", "musician", "Relink Musician")
	client.verifyEmail(app, musicianUser.ID, "relink-musician@example.com", "444444")
	musicianToken := client.login("relink-musician@example.com", "password123")

	client.post("/payout-account/validate", musicianToken, map[string]any{
		"bank_code": "000", "account_number": "0000000000",
	}, http.StatusOK, nil)
	client.post("/payout-account", musicianToken, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "0000000000",
	}, http.StatusOK, nil)

	// A second link for the same PayPetal customer, with a different
	// account — this used to return the "already added" error verbatim and
	// leave user.PayoutAccount unchanged. It should now self-heal.
	client.post("/payout-account/validate", musicianToken, map[string]any{
		"bank_code": "000", "account_number": "1111111111",
	}, http.StatusOK, nil)
	var relinked domain.PayoutAccount
	client.post("/payout-account", musicianToken, map[string]any{
		"bank_code": "000", "bank_name": "Fake Test Bank", "account_number": "1111111111",
	}, http.StatusOK, &relinked)

	if relinked.AccountNumber != "1111111111" {
		t.Fatalf("expected the relink to replace the payout account with the new number, got %#v", relinked)
	}

	var profile domain.User
	client.get("/users/profile", musicianToken, http.StatusOK, &profile)
	if profile.PayoutAccount == nil || profile.PayoutAccount.AccountNumber != "1111111111" {
		t.Fatalf("expected the saved profile to reflect the new payout account, got %#v", profile.PayoutAccount)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
