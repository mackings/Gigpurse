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

func (r *milestoneRepository) GetByID(ctx context.Context, id string) (*domain.Milestone, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, exists := r.milestones[id]
	if !exists {
		return nil, errors.New("milestone not found")
	}
	return m, nil
}

func (r *milestoneRepository) ListByContract(ctx context.Context, contractID string) ([]*domain.Milestone, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.Milestone
	for _, m := range r.milestones {
		if m.ContractID == contractID {
			out = append(out, m)
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
			out = append(out, m)
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

func (r *milestoneRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.milestones, id)
	return nil
}
