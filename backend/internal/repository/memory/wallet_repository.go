package memory

import (
	"context"
	"sync"
	"time"

	"gigpurse/internal/domain"
)

type walletRepository struct {
	mu           sync.RWMutex
	wallets      map[string]*domain.Wallet
	transactions []*domain.Transaction
	nextTxID     int
}

func NewWalletRepository() domain.WalletRepository {
	return &walletRepository{
		wallets: make(map[string]*domain.Wallet),
	}
}

func (r *walletRepository) GetOrCreate(ctx context.Context, userID string) (*domain.Wallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if w, exists := r.wallets[userID]; exists {
		return w, nil
	}
	w := &domain.Wallet{UserID: userID, UpdatedAt: time.Now()}
	r.wallets[userID] = w
	return w, nil
}

func (r *walletRepository) Save(ctx context.Context, wallet *domain.Wallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	wallet.UpdatedAt = time.Now()
	r.wallets[wallet.UserID] = wallet
	return nil
}

func (r *walletRepository) AddTransaction(ctx context.Context, tx *domain.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextTxID++
	if tx.ID == "" {
		tx.ID = time.Now().Format("20060102150405") + "-" + string(rune(r.nextTxID))
	}
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}
	r.transactions = append([]*domain.Transaction{tx}, r.transactions...)
	return nil
}

func (r *walletRepository) ListTransactions(ctx context.Context, userID string) ([]*domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*domain.Transaction
	for _, tx := range r.transactions {
		if tx.UserID == userID {
			out = append(out, tx)
		}
	}
	return out, nil
}
