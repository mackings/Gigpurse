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

type reviewRepository struct {
	collection *mongo.Collection
}

func NewReviewRepository(db *mongo.Database) domain.ReviewRepository {
	return &reviewRepository{
		collection: db.Collection("reviews"),
	}
}

func (r *reviewRepository) Create(ctx context.Context, review *domain.Review) error {
	if review.ID == "" {
		review.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, review)
	return err
}

func (r *reviewRepository) ListByReviewee(ctx context.Context, revieweeID string) ([]*domain.Review, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := r.collection.Find(ctx, bson.M{"reviewee_id": revieweeID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reviews []*domain.Review
	for cursor.Next(ctx) {
		var rev domain.Review
		if err := cursor.Decode(&rev); err != nil {
			return nil, err
		}
		reviews = append(reviews, &rev)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return reviews, nil
}

func (r *reviewRepository) GetByContractAndReviewer(ctx context.Context, contractID, reviewerID string) (*domain.Review, error) {
	var review domain.Review
	err := r.collection.FindOne(ctx, bson.M{"contract_id": contractID, "reviewer_id": reviewerID}).Decode(&review)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("review not found")
		}
		return nil, err
	}
	return &review, nil
}
