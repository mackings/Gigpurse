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
