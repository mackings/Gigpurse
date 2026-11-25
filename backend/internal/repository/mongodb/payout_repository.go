package mongodb

import (
	"context"
	"time"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type payoutRepository struct {
	collection *mongo.Collection
}

func NewPayoutRepository(db *mongo.Database) domain.PayoutRepository {
	return &payoutRepository{collection: db.Collection("payouts")}
}

func (r *payoutRepository) ReplacePending(ctx context.Context, referenceID string, payouts []*domain.Payout) error {
	if _, err := r.collection.DeleteMany(ctx, bson.M{"reference_id": referenceID, "status": "pending"}); err != nil {
		return err
	}
	for _, p := range payouts {
		p.Status = "pending"
		p.CreatedAt, p.UpdatedAt = time.Now(), time.Now()
		// _id is (reference, kind), so this fails if a due or paid row for
		// the same payout already exists.
		if _, err := r.collection.InsertOne(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (r *payoutRepository) GetByID(ctx context.Context, id string) (*domain.Payout, error) {
	var p domain.Payout
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *payoutRepository) ListByStatus(ctx context.Context, status string) ([]*domain.Payout, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status}, options.Find().SetSort(bson.M{"created_at": 1}))
	if err != nil {
		return nil, err
	}
	var out []*domain.Payout
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *payoutRepository) CompareAndSwapStatus(ctx context.Context, id, from, to string) (bool, error) {
	res, err := r.collection.UpdateOne(ctx,
		bson.M{"_id": id, "status": from},
		bson.M{"$set": bson.M{"status": to, "updated_at": time.Now()}},
	)
	if err != nil {
		return false, err
	}
	return res.ModifiedCount == 1, nil
}

func (r *payoutRepository) Update(ctx context.Context, p *domain.Payout) error {
	p.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": p.ID}, p)
	return err
}
