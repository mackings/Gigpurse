package memory

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"gigpurse/internal/domain"
)

type milestoneRepository struct {
	mu         sync.RWMutex
	milestones map[string]*domain.Milestone
	nextID     int
}

func NewMilestoneRepository() domain.MilestoneRepository {
	return &milestoneRepository{milestones: make(map[string]*domain.Milestone)}
}

func (r *milestoneRepository) Create(ctx context.Context, m *domain.Milestone) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	if m.ID == "" {
		m.ID = "ms_" + strconv.Itoa(r.nextID)
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	r.milestones[m.ID] = m
	return nil
}

// GetByID returns a copy, not the pointer stored in the map — this mirrors
// the real MongoDB repository, which always decodes a fresh struct per
// call. Handing back the shared pointer let two concurrent callers' plain
// field reads race against CompareAndSwapStatus's write to that very same
// struct: a false-positive race the real, always-fresh-struct MongoDB
// repository can never produce, caught by -race on the concurrent
// FinalizeFund test.
func (r *milestoneRepository) GetByID(ctx context.Context, id string) (*domain.Milestone, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, exists := r.milestones[id]
	if !exists {
		return nil, errors.New("milestone not found")
	}
	cp := *m
	return &cp, nil
}

func (r *milestoneRepository) ListByContract(ctx context.Context, contractID string) ([]*domain.Milestone, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.Milestone
	for _, m := range r.milestones {
		if m.ContractID == contractID {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *milestoneRepository) ListByStatus(ctx context.Context, status string) ([]*domain.Milestone, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.Milestone
	for _, m := range r.milestones {
		if m.Status == status {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *milestoneRepository) Update(ctx context.Context, m *domain.Milestone) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.milestones[m.ID]; !exists {
		return errors.New("milestone not found")
	}
	m.UpdatedAt = time.Now()
	r.milestones[m.ID] = m
	return nil
}

// CompareAndSwapStatus mirrors the mongodb repository's atomic
// filter-and-set — the mutex here is what makes it a single indivisible
// step, same guarantee MongoDB gives per-document.
func (r *milestoneRepository) CompareAndSwapStatus(ctx context.Context, id, expectedStatus, newStatus string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	m, exists := r.milestones[id]
	if !exists {
		return false, errors.New("milestone not found")
	}
	if m.Status != expectedStatus {
		return false, nil
	}
	m.Status = newStatus
	m.UpdatedAt = time.Now()
	return true, nil
}

func (r *milestoneRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.milestones, id)
	return nil
}
