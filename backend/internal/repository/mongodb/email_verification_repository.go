package mongodb

import (
	"context"
	"errors"
	"time"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type emailVerificationRepository struct {
	collection *mongo.Collection
}

func NewEmailVerificationRepository(db *mongo.Database) domain.EmailVerificationRepository {
	return &emailVerificationRepository{
		collection: db.Collection("email_verification_tokens"),
	}
}

func (r *emailVerificationRepository) Create(ctx context.Context, token *domain.EmailVerificationToken) error {
	if token.ID == "" {
		token.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, token)
	return err
}

func (r *emailVerificationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	var token domain.EmailVerificationToken
	err := r.collection.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("email verification token not found")
		}
		return nil, err
	}
	return &token, nil
}

func (r *emailVerificationRepository) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"used_at": usedAt}})
	return err
}
