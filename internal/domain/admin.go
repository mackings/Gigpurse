package domain

import "context"

type AdminAnalytics struct {
	TotalUsers     int64 `json:"total_users"`
	TotalJobs      int64 `json:"total_jobs"`
	TotalMessages  int64 `json:"total_messages"`
	TotalContracts int64 `json:"total_contracts"`
	TotalDisputes  int64 `json:"total_disputes"`
}

type AdminUsecase interface {
	GetAnalytics(ctx context.Context) (*AdminAnalytics, error)
	ListAllUsers(ctx context.Context) ([]*User, error)
	ListAllJobs(ctx context.Context) ([]*Job, error)
	DeleteJobListing(ctx context.Context, jobID string) error
}
