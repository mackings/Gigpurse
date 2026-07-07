package mongodb

import (
	"context"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type pushSubscriptionRepository struct {
	collection *mongo.Collection
}

func NewPushSubscriptionRepository(db *mongo.Database) domain.PushSubscriptionRepository {
	return &pushSubscriptionRepository{
		collection: db.Collection("push_subscriptions"),
	}
}

func (r *pushSubscriptionRepository) Upsert(ctx context.Context, sub *domain.PushSubscription) error {
	if sub.ID == "" {
		sub.ID = primitive.NewObjectID().Hex()
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"user_id": sub.UserID, "endpoint": sub.Endpoint},
		bson.M{"$set": sub},
		opts,
	)
	return err
}

func (r *pushSubscriptionRepository) ListByUser(ctx context.Context, userID string) ([]*domain.PushSubscription, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subs []*domain.PushSubscription
	for cursor.Next(ctx) {
		var s domain.PushSubscription
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	return subs, cursor.Err()
}

func (r *pushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, userID, endpoint string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"user_id": userID, "endpoint": endpoint})
	return err
}
