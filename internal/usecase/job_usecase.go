package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gigpurse/internal/domain"
)

type jobUsecase struct {
	jobRepo      domain.JobRepository
	userRepo     domain.UserRepository
	contractRepo domain.ContractRepository
	notifRepo    domain.NotificationRepository
}

func NewJobUsecase(
	jobRepo domain.JobRepository,
	userRepo domain.UserRepository,
	contractRepo domain.ContractRepository,
	notifRepo domain.NotificationRepository,
) domain.JobUsecase {
	return &jobUsecase{
		jobRepo:      jobRepo,
		userRepo:     userRepo,
		contractRepo: contractRepo,
		notifRepo:    notifRepo,
	}
}

func (u *jobUsecase) PostJob(ctx context.Context, clientID, title, description, instrument, genre, location string, budget float64) (*domain.Job, error) {
	if title == "" || description == "" || budget <= 0 {
		return nil, errors.New("invalid job details: title, description, and budget are required")
	}

	// Verify client exists and has client role
	user, err := u.userRepo.GetByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("client validation failed: %w", err)
	}
	if user.Role != "client" {
		return nil, errors.New("only clients can post jobs")
	}

	newJob := &domain.Job{
		ClientID:    clientID,
		Title:       title,
		Description: description,
		Budget:      budget,
		Instrument:  instrument,
		Genre:       genre,
		Location:    location,
		Status:      "open",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := u.jobRepo.Create(ctx, newJob); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return newJob, nil
}

func (u *jobUsecase) GetJob(ctx context.Context, id string) (*domain.Job, error) {
	return u.jobRepo.GetByID(ctx, id)
}

func (u *jobUsecase) ListJobs(ctx context.Context, filter domain.JobFilter) ([]*domain.Job, error) {
	jobs, err := u.jobRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if filter.MaxApplications > 0 {
		filtered := jobs[:0]
		for _, job := range jobs {
			apps, err := u.jobRepo.ListApplications(ctx, job.ID)
			if err != nil {
				return nil, err
			}
			if len(apps) <= filter.MaxApplications {
				filtered = append(filtered, job)
			}
		}
		jobs = filtered
	}

	u.sortJobs(ctx, jobs, filter)
	return jobs, nil
}

func (u *jobUsecase) RecommendedJobs(ctx context.Context, musicianID string, limit int) ([]*domain.Job, error) {
	musician, err := u.userRepo.GetByID(ctx, musicianID)
	if err != nil {
		return nil, fmt.Errorf("musician validation failed: %w", err)
	}
	if musician.Role != "musician" {
		return nil, errors.New("only musicians can receive job recommendations")
	}

	filter := domain.JobFilter{Status: "open"}
	if musician.MusicianProfile != nil {
		filter.Genre = musician.MusicianProfile.Genre
		filter.Instrument = musician.MusicianProfile.Instrument
	}
	filter.Location = musician.Location
	filter.SortBy = "relevance"
	filter.MusicianID = musicianID

	jobs, err := u.ListJobs(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		jobs, err = u.ListJobs(ctx, domain.JobFilter{Status: "open", SortBy: "newest"})
		if err != nil {
			return nil, err
		}
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs, nil
}

func (u *jobUsecase) ApplyForJob(ctx context.Context, musicianID, jobID, proposal string, priceBid float64) (*domain.JobApplication, error) {
	if proposal == "" || priceBid <= 0 {
		return nil, errors.New("invalid application details: proposal and price bid are required")
	}

	// Verify musician exists and is a musician
	user, err := u.userRepo.GetByID(ctx, musicianID)
	if err != nil {
		return nil, fmt.Errorf("musician validation failed: %w", err)
	}
	if user.Role != "musician" {
		return nil, errors.New("only musicians can apply for jobs")
	}

	// Verify job exists and is open
	job, err := u.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("job validation failed: %w", err)
	}
	if job.Status != "open" {
		return nil, errors.New("applications are only accepted for open jobs")
	}

	app := &domain.JobApplication{
		JobID:      jobID,
		MusicianID: musicianID,
		Proposal:   proposal,
		PriceBid:   priceBid,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if err := u.jobRepo.CreateApplication(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to submit application: %w", err)
	}

	return app, nil
}

func (u *jobUsecase) ListJobApplications(ctx context.Context, jobID string) ([]*domain.JobApplication, error) {
	return u.jobRepo.ListApplications(ctx, jobID)
}

func (u *jobUsecase) ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*domain.JobApplication, error) {
	return u.jobRepo.ListApplicationsByMusician(ctx, musicianID)
}

func (u *jobUsecase) ListMusicianJobsByStatus(ctx context.Context, musicianID, status string) ([]*domain.Job, error) {
	if status == "pending" {
		apps, err := u.jobRepo.ListApplicationsByMusician(ctx, musicianID)
		if err != nil {
			return nil, err
		}
		var jobs []*domain.Job
		for _, app := range apps {
			if app.Status != "pending" {
				continue
			}
			job, err := u.jobRepo.GetByID(ctx, app.JobID)
			if err == nil && job != nil {
				jobs = append(jobs, job)
			}
		}
		return jobs, nil
	}

	filter := domain.JobFilter{Status: status}
	allJobs, err := u.jobRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	var jobs []*domain.Job
	for _, job := range allJobs {
		if job.MusicianID == musicianID {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func (u *jobUsecase) AcceptApplication(ctx context.Context, clientID, applicationID string) error {
	app, err := u.jobRepo.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	job, err := u.jobRepo.GetByID(ctx, app.JobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	if job.ClientID != clientID {
		return errors.New("unauthorized: only the job creator can accept applications")
	}

	if job.Status != "open" {
		return errors.New("job is no longer open")
	}

	// Update application status
	app.Status = "accepted"
	if err := u.jobRepo.UpdateApplication(ctx, app); err != nil {
		return fmt.Errorf("failed to update application status: %w", err)
	}

	// Update job status and set hired musician
	job.Status = "active"
	job.MusicianID = app.MusicianID
	job.UpdatedAt = time.Now()
	if err := u.jobRepo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Create official Contract record (Contract System)
	contract := &domain.Contract{
		JobID:       job.ID,
		ClientID:    job.ClientID,
		MusicianID:  app.MusicianID,
		Title:       job.Title,
		Description: job.Description,
		Price:       app.PriceBid,
		Source:      "job",
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := u.contractRepo.Create(ctx, contract); err != nil {
		return fmt.Errorf("failed to create contract: %w", err)
	}

	// Reject other applications for the same job
	allApps, err := u.jobRepo.ListApplications(ctx, job.ID)
	if err == nil {
		for _, otherApp := range allApps {
			if otherApp.ID != app.ID && otherApp.Status == "pending" {
				otherApp.Status = "rejected"
				_ = u.jobRepo.UpdateApplication(ctx, otherApp)
			}
		}
	}

	// Send Notifications (Notification & Email Notification Features)
	u.notifyAndEmail(ctx, job.ClientID, "Application Accepted", fmt.Sprintf("You have hired musician '%s' for gig '%s'. Price: $%.2f", app.MusicianID, job.Title, app.PriceBid))
	u.notifyAndEmail(ctx, app.MusicianID, "Bid Accepted", fmt.Sprintf("Congratulations! Your proposal for gig '%s' was accepted. Price: $%.2f", job.Title, app.PriceBid))

	return nil
}

func (u *jobUsecase) sortJobs(ctx context.Context, jobs []*domain.Job, filter domain.JobFilter) {
	desc := filter.SortOrder != "asc"
	switch filter.SortBy {
	case "budget", "price":
		sort.SliceStable(jobs, func(i, j int) bool {
			if desc {
				return jobs[i].Budget > jobs[j].Budget
			}
			return jobs[i].Budget < jobs[j].Budget
		})
	case "applications", "popularity":
		counts := make(map[string]int, len(jobs))
		for _, job := range jobs {
			apps, err := u.jobRepo.ListApplications(ctx, job.ID)
			if err == nil {
				counts[job.ID] = len(apps)
			}
		}
		sort.SliceStable(jobs, func(i, j int) bool {
			if desc {
				return counts[jobs[i].ID] > counts[jobs[j].ID]
			}
			return counts[jobs[i].ID] < counts[jobs[j].ID]
		})
	case "relevance":
		musician, _ := u.userRepo.GetByID(ctx, filter.MusicianID)
		sort.SliceStable(jobs, func(i, j int) bool {
			return relevanceScore(jobs[i], musician) > relevanceScore(jobs[j], musician)
		})
	default:
		sort.SliceStable(jobs, func(i, j int) bool {
			if desc {
				return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
			}
			return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
		})
	}
}

func relevanceScore(job *domain.Job, musician *domain.User) int {
	if musician == nil {
		return 0
	}
	score := 0
	if musician.MusicianProfile != nil {
		if strings.EqualFold(job.Genre, musician.MusicianProfile.Genre) {
			score += 3
		}
		if strings.EqualFold(job.Instrument, musician.MusicianProfile.Instrument) {
			score += 3
		}
	}
	if strings.EqualFold(job.Location, musician.Location) {
		score += 2
	}
	return score
}

func (u *jobUsecase) notifyAndEmail(ctx context.Context, userID, title, message string) {
	notif := &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)
	log.Printf("[EMAIL OUTBOX] To User %s: Subject: %s | Message: %s", userID, title, message)
}
