package mongodb

import (
	"context"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type notificationRepository struct {
	collection *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) domain.NotificationRepository {
	return &notificationRepository{
		collection: db.Collection("notifications"),
	}
}

func (r *notificationRepository) Create(ctx context.Context, notif *domain.Notification) error {
	if notif.ID == "" {
		notif.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, notif)
	return err
}

func (r *notificationRepository) ListForUser(ctx context.Context, userID string) ([]*domain.Notification, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*domain.Notification
	for cursor.Next(ctx) {
		var n domain.Notification
		if err := cursor.Decode(&n); err != nil {
			return nil, err
		}
		notifications = append(notifications, &n)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, notifID, userID string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": notifID, "user_id": userID},
		bson.M{"$set": bson.M{"is_read": true}},
	)
	return err
}
