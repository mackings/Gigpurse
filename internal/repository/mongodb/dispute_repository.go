package mongodb

import (
	"context"
	"errors"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type disputeRepository struct {
	collection *mongo.Collection
}

func NewDisputeRepository(db *mongo.Database) domain.DisputeRepository {
	return &disputeRepository{
		collection: db.Collection("disputes"),
	}
}

func (r *disputeRepository) Create(ctx context.Context, dispute *domain.Dispute) error {
	if dispute.ID == "" {
		dispute.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, dispute)
	return err
}

func (r *disputeRepository) GetByID(ctx context.Context, id string) (*domain.Dispute, error) {
	var dispute domain.Dispute
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&dispute)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("dispute not found")
		}
		return nil, err
	}
	return &dispute, nil
}

func (r *disputeRepository) Update(ctx context.Context, dispute *domain.Dispute) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": dispute.ID}, dispute)
	return err
}

func (r *disputeRepository) List(ctx context.Context, status string) ([]*domain.Dispute, error) {
	query := bson.M{}
	if status != "" {
		query["status"] = status
	}
	return r.find(ctx, query)
}

func (r *disputeRepository) ListForUser(ctx context.Context, userID string) ([]*domain.Dispute, error) {
	return r.find(ctx, bson.M{"$or": []bson.M{
		{"client_id": userID},
		{"musician_id": userID},
		{"opened_by_id": userID},
	}})
}

func (r *disputeRepository) find(ctx context.Context, query bson.M) ([]*domain.Dispute, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var disputes []*domain.Dispute
	for cursor.Next(ctx) {
		var dispute domain.Dispute
		if err := cursor.Decode(&dispute); err != nil {
			return nil, err
		}
		disputes = append(disputes, &dispute)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return disputes, nil
}
