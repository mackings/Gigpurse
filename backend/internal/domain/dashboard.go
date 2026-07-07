package domain

import "context"

type TalentDashboard struct {
	MusicianID          string            `json:"musician_id"`
	PendingApplications []*JobApplication `json:"pending_applications"`
	ActiveJobs          []*Job            `json:"active_jobs"`
	CompletedJobs       []*Job            `json:"completed_jobs"`
	Contracts           []*Contract       `json:"contracts"`
	AverageRating       float64           `json:"average_rating"`
	Reviews             []*Review         `json:"reviews"`
	RecommendedJobs     []*Job            `json:"recommended_jobs"`
}

type DashboardUsecase interface {
	GetTalentDashboard(ctx context.Context, musicianID string) (*TalentDashboard, error)
}
