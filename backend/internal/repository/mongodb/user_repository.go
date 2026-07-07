package mongodb

import (
	"context"
	"errors"
	"strings"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

// musicianAggResult mirrors domain.User plus the fields computed by the
// $lookup/$avg aggregation below. domain.User's AverageRating/TotalReviews
// carry bson:"-" (never persisted), so decoding must happen into this
// wrapper and then be copied across explicitly.
type musicianAggResult struct {
	domain.User   `bson:",inline"`
	AverageRating float64 `bson:"average_rating"`
	TotalReviews  int     `bson:"total_reviews"`
}

func (r *userRepository) ListMusicians(ctx context.Context, filter domain.MusicianFilter) ([]*domain.User, error) {
	// Base query: only list users who are musicians
	query := bson.M{"role": "musician"}

	if filter.Genre != "" {
		query["musician_profile.genres"] = bson.M{"$regex": filter.Genre, "$options": "i"}
	}
	if filter.Instrument != "" {
		query["musician_profile.instruments"] = bson.M{"$regex": filter.Instrument, "$options": "i"}
	}
	if filter.Location != "" {
		query["location"] = bson.M{"$regex": filter.Location, "$options": "i"}
	}
	if filter.MinExp > 0 {
		query["musician_profile.experience_years"] = bson.M{"$gte": filter.MinExp}
	}

	sortOrder := -1
	if filter.SortOrder == "asc" {
		sortOrder = 1
	}

	// average_rating/total_reviews are computed via $lookup on every listing
	// (not just when sorting by rating) so the browse page can render a
	// rating per card without an N+1 follow-up call per musician.
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "reviews",
			"localField":   "_id",
			"foreignField": "reviewee_id",
			"as":           "_reviews",
		}}},
		{{Key: "$addFields", Value: bson.M{
			"average_rating": bson.M{"$ifNull": bson.A{bson.M{"$avg": "$_reviews.rating"}, 0}},
			"total_reviews":  bson.M{"$size": "$_reviews"},
		}}},
		{{Key: "$project", Value: bson.M{"_reviews": 0}}},
	}

	var sortField string
	switch strings.ToLower(filter.SortBy) {
	case "experience":
		sortField = "musician_profile.experience_years"
	case "newest":
		sortField = "created_at"
	case "rating":
		sortField = "average_rating"
	}
	if sortField != "" {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.M{sortField: sortOrder}}})
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*domain.User
	for cursor.Next(ctx) {
		var agg musicianAggResult
		if err := cursor.Decode(&agg); err != nil {
			return nil, err
		}
		user := agg.User
		user.AverageRating = agg.AverageRating
		user.TotalReviews = agg.TotalReviews
		users = append(users, &user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
