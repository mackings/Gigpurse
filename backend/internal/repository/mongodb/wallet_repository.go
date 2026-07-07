package mongodb

import (
	"context"
	"time"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type walletRepository struct {
	wallets      *mongo.Collection
	transactions *mongo.Collection
}

func NewWalletRepository(db *mongo.Database) domain.WalletRepository {
	return &walletRepository{
		wallets:      db.Collection("wallets"),
		transactions: db.Collection("transactions"),
	}
}

func (r *walletRepository) GetOrCreate(ctx context.Context, userID string) (*domain.Wallet, error) {
	var wallet domain.Wallet
	err := r.wallets.FindOne(ctx, bson.M{"user_id": userID}).Decode(&wallet)
	if err == mongo.ErrNoDocuments {
		wallet = domain.Wallet{UserID: userID, UpdatedAt: time.Now()}
		if _, err := r.wallets.InsertOne(ctx, wallet); err != nil {
			return nil, err
		}
		return &wallet, nil
	}
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) Save(ctx context.Context, wallet *domain.Wallet) error {
	wallet.UpdatedAt = time.Now()
	opts := options.Replace().SetUpsert(true)
	_, err := r.wallets.ReplaceOne(ctx, bson.M{"user_id": wallet.UserID}, wallet, opts)
	return err
}

func (r *walletRepository) AddTransaction(ctx context.Context, tx *domain.Transaction) error {
	if tx.ID == "" {
		tx.ID = primitive.NewObjectID().Hex()
	}
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}
	_, err := r.transactions.InsertOne(ctx, tx)
	return err
}

func (r *walletRepository) ListTransactions(ctx context.Context, userID string) ([]*domain.Transaction, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := r.transactions.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var txs []*domain.Transaction
	for cursor.Next(ctx) {
		var tx domain.Transaction
		if err := cursor.Decode(&tx); err != nil {
			return nil, err
		}
		txs = append(txs, &tx)
	}
	return txs, cursor.Err()
}
