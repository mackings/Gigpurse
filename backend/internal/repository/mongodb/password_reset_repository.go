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

type passwordResetRepository struct {
	collection *mongo.Collection
}

func NewPasswordResetRepository(db *mongo.Database) domain.PasswordResetRepository {
	return &passwordResetRepository{
		collection: db.Collection("password_reset_tokens"),
	}
}

func (r *passwordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	if token.ID == "" {
		token.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, token)
	return err
}

func (r *passwordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	var token domain.PasswordResetToken
	err := r.collection.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("password reset token not found")
		}
		return nil, err
	}
	return &token, nil
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"used_at": usedAt}})
	return err
}
