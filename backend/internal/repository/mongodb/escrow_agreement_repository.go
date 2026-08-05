package mongodb

import (
	"context"
	"time"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type escrowAgreementRepository struct {
	collection *mongo.Collection
}

func NewEscrowAgreementRepository(db *mongo.Database) domain.EscrowAgreementRepository {
	return &escrowAgreementRepository{collection: db.Collection("escrow_agreements")}
}

func (r *escrowAgreementRepository) Create(ctx context.Context, a *domain.EscrowAgreement) error {
	if a.ID == "" {
		a.ID = primitive.NewObjectID().Hex()
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	_, err := r.collection.InsertOne(ctx, a)
	return err
}

func (r *escrowAgreementRepository) GetByReference(ctx context.Context, referenceID string) (*domain.EscrowAgreement, error) {
	var a domain.EscrowAgreement
	err := r.collection.FindOne(ctx, bson.M{"reference_id": referenceID}).Decode(&a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *escrowAgreementRepository) Update(ctx context.Context, a *domain.EscrowAgreement) error {
	a.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": a.ID}, a)
	return err
}

func (r *escrowAgreementRepository) ListStalePending(ctx context.Context, olderThan time.Time) ([]*domain.EscrowAgreement, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"status":     "PENDING",
		"created_at": bson.M{"$lt": olderThan},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var agreements []*domain.EscrowAgreement
	for cursor.Next(ctx) {
		var a domain.EscrowAgreement
		if err := cursor.Decode(&a); err != nil {
			return nil, err
		}
		agreements = append(agreements, &a)
	}
	return agreements, cursor.Err()
}
