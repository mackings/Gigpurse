package memory

import (
	"context"
	"sync"
	"testing"

	"gigpurse/internal/domain"
)

// TestCompareAndSwapStatus_ConcurrentCallers reproduces the exact race that
// slipped past a plain read-then-write status check in production: many
// callers (standing in for PayPetal's webhook and GigPurse's own
// reconciler both finalizing the same payment) all attempt the same
// "accepted" -> "funded" transition at once. Exactly one must win; every
// loser must see false with no error, and the milestone must end up
// "funded" exactly once, never silently reverted.
func TestCompareAndSwapStatus_ConcurrentCallers(t *testing.T) {
	repo := NewMilestoneRepository()
	ctx := context.Background()

	m := &domain.Milestone{Status: "accepted"}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}

	const attempts = 50
	var wg sync.WaitGroup
	wins := make([]bool, attempts)
	errs := make([]error, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			won, err := repo.CompareAndSwapStatus(ctx, m.ID, "accepted", "funded")
			wins[i] = won
			errs[i] = err
		}(i)
	}
	wg.Wait()

	winCount := 0
	for i, w := range wins {
		if errs[i] != nil {
			t.Fatalf("attempt %d returned an error: %v", i, errs[i])
		}
		if w {
			winCount++
		}
	}
	if winCount != 1 {
		t.Fatalf("expected exactly 1 winner among %d concurrent callers, got %d", attempts, winCount)
	}

	final, err := repo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if final.Status != "funded" {
		t.Fatalf("expected final status \"funded\", got %q", final.Status)
	}
}

// TestCompareAndSwapStatus_LoserDoesNotBlock verifies the losing calls
// don't error out — a caller (like FinalizeFund) treats false as "someone
// else already handled this," not a failure.
func TestCompareAndSwapStatus_LoserDoesNotBlock(t *testing.T) {
	repo := NewMilestoneRepository()
	ctx := context.Background()

	m := &domain.Milestone{Status: "accepted"}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}

	won, err := repo.CompareAndSwapStatus(ctx, m.ID, "accepted", "funded")
	if err != nil || !won {
		t.Fatalf("first swap should win cleanly: won=%v err=%v", won, err)
	}

	won, err = repo.CompareAndSwapStatus(ctx, m.ID, "accepted", "funded")
	if err != nil {
		t.Fatalf("losing swap should not error: %v", err)
	}
	if won {
		t.Fatal("second swap should lose — status is already \"funded\", not \"accepted\"")
	}
}
