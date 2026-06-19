package usecase

import (
	"context"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type adminUsecase struct {
	db       *mongo.Database
	userRepo domain.UserRepository
	jobRepo  domain.JobRepository
}

func NewAdminUsecase(db *mongo.Database, ur domain.UserRepository, jr domain.JobRepository) domain.AdminUsecase {
	return &adminUsecase{
		db:       db,
		userRepo: ur,
		jobRepo:  jr,
	}
}

func (u *adminUsecase) GetAnalytics(ctx context.Context) (*domain.AdminAnalytics, error) {
	totalUsers, err := u.db.Collection("users").CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	totalJobs, err := u.db.Collection("jobs").CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	totalMessages, err := u.db.Collection("chat_messages").CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	totalContracts, err := u.db.Collection("contracts").CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	totalDisputes, err := u.db.Collection("disputes").CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	return &domain.AdminAnalytics{
		TotalUsers:     totalUsers,
		TotalJobs:      totalJobs,
		TotalMessages:  totalMessages,
		TotalContracts: totalContracts,
		TotalDisputes:  totalDisputes,
	}, nil
}

func (u *adminUsecase) ListAllUsers(ctx context.Context) ([]*domain.User, error) {
	cursor, err := u.db.Collection("users").Find(ctx, bson.M{})
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

	return users, nil
}

func (u *adminUsecase) ListAllJobs(ctx context.Context) ([]*domain.Job, error) {
	return u.jobRepo.List(ctx, domain.JobFilter{})
}

func (u *adminUsecase) DeleteJobListing(ctx context.Context, jobID string) error {
	_, err := u.db.Collection("jobs").DeleteOne(ctx, bson.M{"_id": jobID})
	return err
}
