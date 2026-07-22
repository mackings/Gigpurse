package mongodb

import (
	"context"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type chatRepository struct {
	collection *mongo.Collection
}

func NewChatRepository(db *mongo.Database) domain.ChatRepository {
	return &chatRepository{
		collection: db.Collection("chat_messages"),
	}
}

func (r *chatRepository) SaveMessage(ctx context.Context, msg *domain.ChatMessage) error {
	if msg.ID == "" {
		msg.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, msg)
	return err
}

func (r *chatRepository) GetChatHistory(ctx context.Context, user1, user2 string) ([]*domain.ChatMessage, error) {
	query := bson.M{
		// Dispute-room messages share this collection but aren't part of any
		// 1:1 pairwise history — they're addressed by dispute_id, not
		// recv_id, and would otherwise leak in here as empty-recv_id noise.
		"dispute_id": bson.M{"$in": bson.A{nil, ""}},
		"$or": []bson.M{
			{"sender_id": user1, "recv_id": user2},
			{"sender_id": user2, "recv_id": user1},
		},
	}

	opts := options.Find().SetSort(bson.M{"timestamp": 1}) // ASC order

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Initialized (not nil) so an empty result serializes as `[]`, not
	// `null` — frontend code checks messages.length, which silently breaks
	// against null (e.g. the "is this a brand-new conversation?" check).
	messages := []*domain.ChatMessage{}
	for cursor.Next(ctx) {
		var msg domain.ChatMessage
		if err := cursor.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *chatRepository) ListByDispute(ctx context.Context, disputeID string) ([]*domain.ChatMessage, error) {
	opts := options.Find().SetSort(bson.M{"timestamp": 1})
	cursor, err := r.collection.Find(ctx, bson.M{"dispute_id": disputeID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	messages := []*domain.ChatMessage{}
	for cursor.Next(ctx) {
		var msg domain.ChatMessage
		if err := cursor.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, cursor.Err()
}

func (r *chatRepository) GetRecentChats(ctx context.Context, userID string) ([]*domain.ChatMessage, error) {
	// Aggregation to find latest message for each distinct chat partner
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"dispute_id": bson.M{"$in": bson.A{nil, ""}},
			"$or": []bson.M{
				{"sender_id": userID},
				{"recv_id": userID},
			},
		}}},
		{{Key: "$sort", Value: bson.M{"timestamp": -1}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.M{
				"$cond": bson.A{
					bson.M{"$eq": []interface{}{"$sender_id", userID}},
					"$recv_id",
					"$sender_id",
				},
			}},
			{Key: "latest_message", Value: bson.M{"$first": "$$ROOT"}},
		}}},
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$latest_message"}}},
		{{Key: "$sort", Value: bson.M{"timestamp": -1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Initialized (not nil) so an empty result serializes as `[]`, not
	// `null` — frontend code checks messages.length, which silently breaks
	// against null (e.g. the "is this a brand-new conversation?" check).
	messages := []*domain.ChatMessage{}
	for cursor.Next(ctx) {
		var msg domain.ChatMessage
		if err := cursor.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
