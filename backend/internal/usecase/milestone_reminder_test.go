package usecase

import (
	"context"
	"testing"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/repository/memory"
)

type reminderContractRepo struct{ domain.ContractRepository }

func (reminderContractRepo) GetByID(ctx context.Context, id string) (*domain.Contract, error) {
	return &domain.Contract{ID: id, ClientID: "client", MusicianID: "talent"}, nil
}

// An unanswered milestone used to be re-emailed every few minutes forever.
func TestReminders_FirstNudgeThenOncePerDay(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMilestoneRepository()
	m := &domain.Milestone{ContractID: "c1", Title: "Set", Amount: 500, Status: "proposed", ProposedBy: "client"}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatal(err)
	}
	u := &milestoneUsecase{
		paypetalDeps: paypetalDeps{userRepo: &testUserRepo{byID: map[string]*domain.User{
			"talent": {ID: "talent", Email: "talent@example.com"},
		}}},
		milestoneRepo: repo,
		contractRepo:  reminderContractRepo{},
		notifRepo:     &testNotifRepo{},
	}
	// The repo stamps UpdatedAt on every write, so backdate afterwards: GetByID
	// copies, but Create stored this very pointer.
	const nudge, repeat = 5 * time.Minute, 24 * time.Hour
	age := func(d time.Duration) {
		cur, _ := repo.GetByID(ctx, m.ID)
		past := time.Now().Add(-d)
		if cur.LastReminderAt != nil {
			cur.LastReminderAt = &past
		}
		_ = repo.Update(ctx, cur)
		cur.UpdatedAt = past // repo now holds cur itself
	}

	withCapturedEmails(func(capture func() []capturedEmail) {
		for _, step := range []struct {
			age  time.Duration
			want int
		}{
			{time.Minute, 0},
			{6 * time.Minute, 1},
			{6 * time.Minute, 1},
			{23 * time.Hour, 1},
			{25 * time.Hour, 2},
			{time.Hour, 2},
		} {
			age(step.age)
			u.sendDueReminders(ctx, nudge, repeat)
			if got := len(capture()); got != step.want {
				t.Fatalf("after %s: expected %d emails in total, got %d", step.age, step.want, got)
			}
		}
	})
}

// Nobody but the client can release TrustVault money, so an unanswered
// request is chased daily — and never released automatically.
func TestReleaseNudges(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMilestoneRepository()
	ago := func(d time.Duration) *time.Time { v := time.Now().Add(-d); return &v }
	mk := func(title string, asked *time.Time, provider string) *domain.Milestone {
		m := &domain.Milestone{ContractID: "c1", Title: title, Status: "funded", ReleaseRequestedAt: asked, EscrowProvider: provider}
		_ = repo.Create(ctx, m)
		return m
	}
	late := mk("late", ago(49*time.Hour), domain.ProviderVault)
	mk("early", ago(47*time.Hour), domain.ProviderVault)
	mk("unasked", nil, domain.ProviderVault)
	mk("old", ago(90*time.Hour), "")
	u := &milestoneUsecase{
		paypetalDeps: paypetalDeps{userRepo: &testUserRepo{byID: map[string]*domain.User{
			"client": {ID: "client", Email: "client@example.com"}, "talent": {ID: "talent", Email: "talent@example.com"},
		}}},
		milestoneRepo: repo, contractRepo: reminderContractRepo{}, notifRepo: &testNotifRepo{},
	}
	withCapturedEmails(func(capture func() []capturedEmail) {
		u.nudgeOverdueReleases(ctx)
		u.nudgeOverdueReleases(ctx)
		if got := capture(); len(got) != 2 || got[0].to != "client@example.com" || got[1].to != "talent@example.com" {
			t.Fatalf("expected one email each, got %#v", got)
		}
		m, _ := repo.GetByID(ctx, late.ID)
		m.ReleaseNudgedAt = ago(25 * time.Hour)
		_ = repo.Update(ctx, m)
		u.nudgeOverdueReleases(ctx)
		if got := capture(); len(got) != 3 || got[2].to != "client@example.com" {
			t.Fatalf("expected only the client chased again, got %#v", got)
		}
		if m, _ := repo.GetByID(ctx, late.ID); m.Status != "funded" {
			t.Fatalf("must never release by itself, got %q", m.Status)
		}
	})
}
