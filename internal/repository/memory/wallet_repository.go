package memory

import (
	"context"
	"errors"
	"sync"

	"gigpurse/internal/domain"
)

type walletRepository struct {
	mu      sync.RWMutex
	wallets map[string]*domain.Wallet
}

func NewWalletRepository() domain.WalletRepository {
	return &walletRepository{
		wallets: make(map[string]*domain.Wallet),
	}
}

func (r *walletRepository) GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, w := range r.wallets {
		if w.UserID == userID {
			return w, nil
		}
	}
	return nil, errors.New("wallet not found")
}

func (r *walletRepository) UpdateBalance(ctx context.Context, walletID string, amount float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, exists := r.wallets[walletID]
	if !exists {
		return errors.New("wallet not found")
	}

	w.Balance += amount
	return nil
}

func (r *walletRepository) Create(ctx context.Context, wallet *domain.Wallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.wallets[wallet.ID]; exists {
		return errors.New("wallet already exists")
	}

	r.wallets[wallet.ID] = wallet
	return nil
}
