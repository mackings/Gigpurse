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

type milestoneRepository struct {
	collection *mongo.Collection
}

func NewMilestoneRepository(db *mongo.Database) domain.MilestoneRepository {
	return &milestoneRepository{
		collection: db.Collection("milestones"),
	}
}

func (r *milestoneRepository) Create(ctx context.Context, m *domain.Milestone) error {
	if m.ID == "" {
		m.ID = primitive.NewObjectID().Hex()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	_, err := r.collection.InsertOne(ctx, m)
	return err
}

func (r *milestoneRepository) GetByID(ctx context.Context, id string) (*domain.Milestone, error) {
	var m domain.Milestone
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *milestoneRepository) ListByContract(ctx context.Context, contractID string) ([]*domain.Milestone, error) {
	opts := options.Find().SetSort(bson.M{"order": 1})
	cursor, err := r.collection.Find(ctx, bson.M{"contract_id": contractID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var milestones []*domain.Milestone
	for cursor.Next(ctx) {
		var m domain.Milestone
		if err := cursor.Decode(&m); err != nil {
			return nil, err
		}
		milestones = append(milestones, &m)
	}
	return milestones, cursor.Err()
}

func (r *milestoneRepository) ListByStatus(ctx context.Context, status string) ([]*domain.Milestone, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var milestones []*domain.Milestone
	for cursor.Next(ctx) {
		var m domain.Milestone
		if err := cursor.Decode(&m); err != nil {
			return nil, err
		}
		milestones = append(milestones, &m)
	}
	return milestones, cursor.Err()
}

func (r *milestoneRepository) Update(ctx context.Context, m *domain.Milestone) error {
	m.UpdatedAt = time.Now()
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": m.ID}, m)
	return err
}

func (r *milestoneRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
