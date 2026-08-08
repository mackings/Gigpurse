package memory

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"gigpurse/internal/domain"
)

type escrowAgreementRepository struct {
	mu         sync.RWMutex
	agreements map[string]*domain.EscrowAgreement
	nextID     int
}

func NewEscrowAgreementRepository() domain.EscrowAgreementRepository {
	return &escrowAgreementRepository{agreements: make(map[string]*domain.EscrowAgreement)}
}

func (r *escrowAgreementRepository) Create(ctx context.Context, a *domain.EscrowAgreement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	if a.ID == "" {
		a.ID = "esc_" + strconv.Itoa(r.nextID)
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	r.agreements[a.ID] = a
	return nil
}

func (r *escrowAgreementRepository) GetByReference(ctx context.Context, referenceID string) (*domain.EscrowAgreement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, a := range r.agreements {
		if a.ReferenceID == referenceID {
			return a, nil
		}
	}
	return nil, errors.New("escrow agreement not found")
}

func (r *escrowAgreementRepository) Update(ctx context.Context, a *domain.EscrowAgreement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agreements[a.ID]; !exists {
		return errors.New("escrow agreement not found")
	}
	a.UpdatedAt = time.Now()
	r.agreements[a.ID] = a
	return nil
}

func (r *escrowAgreementRepository) ListByInitiator(ctx context.Context, userID string) ([]*domain.EscrowAgreement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.EscrowAgreement
	for _, a := range r.agreements {
		if a.InitiatorUserID == userID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *escrowAgreementRepository) ListPending(ctx context.Context) ([]*domain.EscrowAgreement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.EscrowAgreement
	for _, a := range r.agreements {
		if a.Status == "PENDING" {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *escrowAgreementRepository) ListStalePending(ctx context.Context, olderThan time.Time) ([]*domain.EscrowAgreement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.EscrowAgreement
	for _, a := range r.agreements {
		if a.Status == "PENDING" && a.CreatedAt.Before(olderThan) {
			out = append(out, a)
		}
	}
	return out, nil
}
