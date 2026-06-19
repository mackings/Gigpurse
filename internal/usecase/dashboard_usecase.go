package usecase

import (
	"context"

	"gigpurse/internal/domain"
)

type dashboardUsecase struct {
	jobUsecase      domain.JobUsecase
	contractUsecase domain.ContractUsecase
	reviewUsecase   domain.ReviewUsecase
}

func NewDashboardUsecase(
	jobUsecase domain.JobUsecase,
	contractUsecase domain.ContractUsecase,
	reviewUsecase domain.ReviewUsecase,
) domain.DashboardUsecase {
	return &dashboardUsecase{
		jobUsecase:      jobUsecase,
		contractUsecase: contractUsecase,
		reviewUsecase:   reviewUsecase,
	}
}

func (u *dashboardUsecase) GetTalentDashboard(ctx context.Context, musicianID string) (*domain.TalentDashboard, error) {
	apps, err := u.jobUsecase.ListApplicationsByMusician(ctx, musicianID)
	if err != nil {
		return nil, err
	}
	var pendingApps []*domain.JobApplication
	for _, app := range apps {
		if app.Status == "pending" {
			pendingApps = append(pendingApps, app)
		}
	}

	activeJobs, err := u.jobUsecase.ListMusicianJobsByStatus(ctx, musicianID, "active")
	if err != nil {
		return nil, err
	}
	completedJobs, err := u.jobUsecase.ListMusicianJobsByStatus(ctx, musicianID, "completed")
	if err != nil {
		return nil, err
	}
	contracts, err := u.contractUsecase.ListUserContracts(ctx, musicianID, "musician")
	if err != nil {
		return nil, err
	}
	reviews, err := u.reviewUsecase.GetUserReviews(ctx, musicianID)
	if err != nil {
		return nil, err
	}
	avg, err := u.reviewUsecase.GetAverageRating(ctx, musicianID)
	if err != nil {
		return nil, err
	}
	recommended, err := u.jobUsecase.RecommendedJobs(ctx, musicianID, 10)
	if err != nil {
		return nil, err
	}

	return &domain.TalentDashboard{
		MusicianID:          musicianID,
		PendingApplications: pendingApps,
		ActiveJobs:          activeJobs,
		CompletedJobs:       completedJobs,
		Contracts:           contracts,
		AverageRating:       avg,
		Reviews:             reviews,
		RecommendedJobs:     recommended,
	}, nil
}
