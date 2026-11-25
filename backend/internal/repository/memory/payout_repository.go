package memory

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"gigpurse/internal/domain"
)

type payoutRepository struct {
	mu   sync.Mutex
	rows map[string]domain.Payout
}

func NewPayoutRepository() domain.PayoutRepository {
	return &payoutRepository{rows: map[string]domain.Payout{}}
}

func (r *payoutRepository) ReplacePending(ctx context.Context, referenceID string, payouts []*domain.Payout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, p := range r.rows {
		if p.ReferenceID == referenceID && p.Status == "pending" {
			delete(r.rows, id)
		}
	}
	for _, p := range payouts {
		if _, exists := r.rows[p.ID]; exists {
			return errors.New("payout already recorded")
		}
		p.Status = "pending"
		p.CreatedAt, p.UpdatedAt = time.Now(), time.Now()
		r.rows[p.ID] = *p
	}
	return nil
}

func (r *payoutRepository) GetByID(ctx context.Context, id string) (*domain.Payout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.rows[id]
	if !ok {
		return nil, errors.New("payout not found")
	}
	return &p, nil
}

func (r *payoutRepository) ListByStatus(ctx context.Context, status string) ([]*domain.Payout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.Payout
	for _, p := range r.rows {
		if p.Status == status {
			cp := p
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *payoutRepository) CompareAndSwapStatus(ctx context.Context, id, from, to string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.rows[id]
	if !ok {
		return false, errors.New("payout not found")
	}
	if p.Status != from {
		return false, nil
	}
	p.Status, p.UpdatedAt = to, time.Now()
	r.rows[id] = p
	return true, nil
}

func (r *payoutRepository) Update(ctx context.Context, p *domain.Payout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[p.ID]; !ok {
		return errors.New("payout not found")
	}
	p.UpdatedAt = time.Now()
	r.rows[p.ID] = *p
	return nil
}
