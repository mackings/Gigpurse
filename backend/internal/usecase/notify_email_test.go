package usecase

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"gigpurse/internal/domain"
)

// testNotifRepo/testUserRepo/testChatRepo are minimal, purpose-built test
// doubles — not full CRUD fakes — since notify()/sendUnreadDigests() only
// ever touch a handful of methods on each interface. No reusable fake for
// these exists outside api_simulation_test.go (package http_test, not
// reachable from here), so these stay local to this file.

type testNotifRepo struct {
	mu      sync.Mutex
	created []*domain.Notification
}

func (r *testNotifRepo) Create(ctx context.Context, notif *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, notif)
	return nil
}
func (r *testNotifRepo) ListForUser(ctx context.Context, userID string) ([]*domain.Notification, error) {
	return nil, errors.New("not implemented")
}
func (r *testNotifRepo) MarkAsRead(ctx context.Context, notifID, userID string) error {
	return errors.New("not implemented")
}

type testUserRepo struct {
	byID map[string]*domain.User
}

func (r *testUserRepo) Create(ctx context.Context, user *domain.User) error {
	return errors.New("not implemented")
}
func (r *testUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}
func (r *testUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (r *testUserRepo) Update(ctx context.Context, user *domain.User) error {
	return errors.New("not implemented")
}
func (r *testUserRepo) ListMusicians(ctx context.Context, filter domain.MusicianFilter) ([]*domain.User, error) {
	return nil, errors.New("not implemented")
}

// capturedEmail records one sendEmailFn invocation.
type capturedEmail struct {
	to, subject, body string
}

// withCapturedEmails swaps sendEmailFn for the duration of fn, restoring the
// original afterward — every test in this file that checks email dispatch
// uses this instead of asserting on log output.
func withCapturedEmails(fn func(capture func() []capturedEmail)) {
	var mu sync.Mutex
	var sent []capturedEmail
	original := sendEmailFn
	sendEmailFn = func(to, subject, body string) error {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, capturedEmail{to, subject, body})
		return nil
	}
	defer func() { sendEmailFn = original }()
	fn(func() []capturedEmail {
		mu.Lock()
		defer mu.Unlock()
		return append([]capturedEmail{}, sent...)
	})
}

func TestMilestoneNotify_SendsRealEmail(t *testing.T) {
	notifRepo := &testNotifRepo{}
	userRepo := &testUserRepo{byID: map[string]*domain.User{
		"user-1": {ID: "user-1", Email: "musician@example.com", Name: "Test Musician"},
	}}
	u := &milestoneUsecase{
		paypetalDeps: paypetalDeps{userRepo: userRepo},
		notifRepo:    notifRepo,
	}

	withCapturedEmails(func(capture func() []capturedEmail) {
		u.notify(context.Background(), "user-1", "Escrow funded", "Escrow funded for milestone 'Rehearsal' (₦10,000).", "contract-1")

		if len(notifRepo.created) != 1 {
			t.Fatalf("expected 1 in-app notification, got %d", len(notifRepo.created))
		}
		if notifRepo.created[0].UserID != "user-1" || notifRepo.created[0].Title != "Escrow funded" {
			t.Fatalf("unexpected notification: %#v", notifRepo.created[0])
		}

		sent := capture()
		if len(sent) != 1 {
			t.Fatalf("expected 1 real email sent, got %d", len(sent))
		}
		if sent[0].to != "musician@example.com" {
			t.Fatalf("expected email to resolved address musician@example.com, got %q", sent[0].to)
		}
		if sent[0].subject != "Escrow funded" || !strings.Contains(sent[0].body, "Rehearsal") {
			t.Fatalf("unexpected email content: %#v", sent[0])
		}
	})
}

func TestMilestoneNotify_SkipsEmailWhenUserHasNoAddress(t *testing.T) {
	notifRepo := &testNotifRepo{}
	userRepo := &testUserRepo{byID: map[string]*domain.User{
		"user-1": {ID: "user-1", Email: "", Name: "No Email User"},
	}}
	u := &milestoneUsecase{
		paypetalDeps: paypetalDeps{userRepo: userRepo},
		notifRepo:    notifRepo,
	}

	withCapturedEmails(func(capture func() []capturedEmail) {
		u.notify(context.Background(), "user-1", "Title", "Message", "contract-1")

		if len(notifRepo.created) != 1 {
			t.Fatalf("expected the in-app notification to still be created, got %d", len(notifRepo.created))
		}
		if sent := capture(); len(sent) != 0 {
			t.Fatalf("expected no email attempted for a user with no address, got %#v", sent)
		}
	})
}

// TestContractNotifyAndEmail_ResolvesRecipientEmail specifically guards the
// bug found while wiring this up: notifyAndEmail used to log "To User
// %s" with the raw userID (never even fetching the user), not a real email
// address — this asserts the fix actually resolves and uses user.Email.
func TestContractNotifyAndEmail_ResolvesRecipientEmail(t *testing.T) {
	notifRepo := &testNotifRepo{}
	userRepo := &testUserRepo{byID: map[string]*domain.User{
		"client-1": {ID: "client-1", Email: "client@example.com", Name: "Test Client"},
	}}
	u := &contractUsecase{
		paypetalDeps: paypetalDeps{userRepo: userRepo},
		notifRepo:    notifRepo,
	}

	withCapturedEmails(func(capture func() []capturedEmail) {
		u.notifyAndEmail(context.Background(), "client-1", "Booking Accepted", "Your booking was accepted.", "/messages")

		sent := capture()
		if len(sent) != 1 {
			t.Fatalf("expected 1 real email sent, got %d", len(sent))
		}
		if sent[0].to != "client@example.com" {
			t.Fatalf("expected the resolved email address, got %q (this is the raw-userID bug if it says client-1)", sent[0].to)
		}
	})
}

// testChatRepoForDigest stubs just enough of domain.ChatRepository for
// sendUnreadDigests: a fixed set of "unread" messages, and a place to
// record which IDs got marked as reminded.
type testChatRepoForDigest struct {
	unread        []*domain.ChatMessage
	remindedIDs   []string
	markReadCalls int
}

func (r *testChatRepoForDigest) SaveMessage(ctx context.Context, msg *domain.ChatMessage) error {
	return errors.New("not implemented")
}
func (r *testChatRepoForDigest) GetChatHistory(ctx context.Context, user1, user2 string) ([]*domain.ChatMessage, error) {
	return nil, errors.New("not implemented")
}
func (r *testChatRepoForDigest) GetRecentChats(ctx context.Context, userID string) ([]*domain.ChatMessage, error) {
	return nil, errors.New("not implemented")
}
func (r *testChatRepoForDigest) ListByDispute(ctx context.Context, disputeID string) ([]*domain.ChatMessage, error) {
	return nil, errors.New("not implemented")
}
func (r *testChatRepoForDigest) MarkConversationRead(ctx context.Context, recvID, senderID string) error {
	r.markReadCalls++
	return nil
}
func (r *testChatRepoForDigest) ListUnreadOlderThan(ctx context.Context, cutoff time.Time) ([]*domain.ChatMessage, error) {
	return r.unread, nil
}
func (r *testChatRepoForDigest) MarkReminderEmailSent(ctx context.Context, ids []string) error {
	r.remindedIDs = append(r.remindedIDs, ids...)
	return nil
}

// TestChatUsecase_SendUnreadDigests_OnePerRecipient is the core behavior
// the whole feature exists for: a recipient with multiple stale-unread
// messages (even from different senders) gets exactly one digest email,
// not one per message — and every covered message ID gets marked so the
// next scan doesn't re-send.
func TestChatUsecase_SendUnreadDigests_OnePerRecipient(t *testing.T) {
	old := time.Now().Add(-48 * time.Hour)
	repo := &testChatRepoForDigest{unread: []*domain.ChatMessage{
		{ID: "msg-1", SenderID: "sender-a", RecvID: "recv-1", Timestamp: old},
		{ID: "msg-2", SenderID: "sender-b", RecvID: "recv-1", Timestamp: old},
		{ID: "msg-3", SenderID: "sender-a", RecvID: "recv-2", Timestamp: old},
	}}
	userRepo := &testUserRepo{byID: map[string]*domain.User{
		"recv-1":   {ID: "recv-1", Email: "recv1@example.com", Name: "Recipient One"},
		"recv-2":   {ID: "recv-2", Email: "recv2@example.com", Name: "Recipient Two"},
		"sender-a": {ID: "sender-a", Email: "a@example.com", Name: "Sender A"},
		"sender-b": {ID: "sender-b", Email: "b@example.com", Name: "Sender B"},
	}}
	u := &chatUsecase{chatRepo: repo, userRepo: userRepo}

	withCapturedEmails(func(capture func() []capturedEmail) {
		u.sendUnreadDigests(context.Background(), 24*time.Hour)

		sent := capture()
		if len(sent) != 2 {
			t.Fatalf("expected exactly 2 digest emails (one per recipient), got %d: %#v", len(sent), sent)
		}
		byTo := map[string]capturedEmail{}
		for _, e := range sent {
			byTo[e.to] = e
		}
		recv1Email, ok := byTo["recv1@example.com"]
		if !ok {
			t.Fatalf("expected a digest email to recv1@example.com, got %#v", sent)
		}
		if !strings.Contains(recv1Email.body, "Sender A") || !strings.Contains(recv1Email.body, "Sender B") {
			t.Fatalf("expected recv-1's single digest to mention both senders, got body: %q", recv1Email.body)
		}
		if _, ok := byTo["recv2@example.com"]; !ok {
			t.Fatalf("expected a digest email to recv2@example.com, got %#v", sent)
		}

		if len(repo.remindedIDs) != 3 {
			t.Fatalf("expected all 3 covered message IDs marked as reminded, got %v", repo.remindedIDs)
		}
	})
}

func TestChatUsecase_SendUnreadDigests_NoOpWhenNothingStale(t *testing.T) {
	repo := &testChatRepoForDigest{unread: nil}
	u := &chatUsecase{chatRepo: repo, userRepo: &testUserRepo{byID: map[string]*domain.User{}}}

	withCapturedEmails(func(capture func() []capturedEmail) {
		u.sendUnreadDigests(context.Background(), 24*time.Hour)
		if sent := capture(); len(sent) != 0 {
			t.Fatalf("expected no emails when there's nothing stale, got %#v", sent)
		}
	})
}
