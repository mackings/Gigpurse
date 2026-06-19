package mongodb

import (
	"context"
	"errors"
	"strings"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type userRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) domain.UserRepository {
	return &userRepository{
		collection: db.Collection("users"),
	}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": user.ID}, user)
	return err
}

func (r *userRepository) ListMusicians(ctx context.Context, filter domain.MusicianFilter) ([]*domain.User, error) {
	// Base query: only list users who are musicians
	query := bson.M{"role": "musician"}

	if filter.Genre != "" {
		query["musician_profile.genre"] = bson.M{"$regex": filter.Genre, "$options": "i"}
	}
	if filter.Instrument != "" {
		query["musician_profile.instrument"] = bson.M{"$regex": filter.Instrument, "$options": "i"}
	}
	if filter.Location != "" {
		query["location"] = bson.M{"$regex": filter.Location, "$options": "i"}
	}
	if filter.MinExp > 0 {
		query["musician_profile.experience_years"] = bson.M{"$gte": filter.MinExp}
	}

	opts := options.Find()
	sortOrder := -1
	if filter.SortOrder == "asc" {
		sortOrder = 1
	}
	switch strings.ToLower(filter.SortBy) {
	case "experience":
		opts.SetSort(bson.M{"musician_profile.experience_years": sortOrder})
	case "newest":
		opts.SetSort(bson.M{"created_at": sortOrder})
	}

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*domain.User
	for cursor.Next(ctx) {
		var user domain.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
