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
	walletRepo   domain.WalletRepository
	reviewRepo   domain.ReviewRepository
}

func NewJobUsecase(
	jobRepo domain.JobRepository,
	userRepo domain.UserRepository,
	contractRepo domain.ContractRepository,
	notifRepo domain.NotificationRepository,
	walletRepo domain.WalletRepository,
	reviewRepo domain.ReviewRepository,
) domain.JobUsecase {
	return &jobUsecase{
		jobRepo:      jobRepo,
		userRepo:     userRepo,
		contractRepo: contractRepo,
		notifRepo:    notifRepo,
		walletRepo:   walletRepo,
		reviewRepo:   reviewRepo,
	}
}

func (u *jobUsecase) PostJob(ctx context.Context, clientID string, input domain.JobPostInput) (*domain.Job, error) {
	if input.Title == "" || input.Description == "" || input.Budget <= 0 {
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
		ClientID:        clientID,
		Title:           input.Title,
		Description:     input.Description,
		Budget:          input.Budget,
		Instrument:      input.Instrument,
		Genre:           input.Genre,
		Location:        input.Location,
		ExperienceLevel: input.ExperienceLevel,
		Duration:        input.Duration,
		ProjectType:     input.ProjectType,
		Skills:          input.Skills,
		// Jobs stay invisible to talent until the client funds escrow —
		// see FundEscrow, which is what flips this to "open".
		Status:    "pending_funding",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := u.jobRepo.Create(ctx, newJob); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return newJob, nil
}

// UpdateJob lets the client edit their own posting after it's live —
// budget is off-limits once escrow is funded, since that amount is already
// locked based on the original figure and changing it here wouldn't move
// any real money. Every musician with a still-pending application is
// notified, since the posting they applied to just changed under them.
func (u *jobUsecase) UpdateJob(ctx context.Context, clientID, jobID string, input domain.JobPostInput) (*domain.Job, error) {
	if input.Title == "" || input.Description == "" || input.Budget <= 0 {
		return nil, errors.New("invalid job details: title, description, and budget are required")
	}
	job, err := u.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	if job.ClientID != clientID {
		return nil, errors.New("unauthorized: only the job's creator can edit it")
	}
	if job.EscrowFunded && input.Budget != job.Budget {
		return nil, errors.New("budget can't be changed after escrow is funded")
	}

	job.Title = input.Title
	job.Description = input.Description
	job.Instrument = input.Instrument
	job.Genre = input.Genre
	job.Location = input.Location
	job.Budget = input.Budget
	job.ExperienceLevel = input.ExperienceLevel
	job.Duration = input.Duration
	job.ProjectType = input.ProjectType
	job.Skills = input.Skills
	job.UpdatedAt = time.Now()
	if err := u.jobRepo.Update(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	u.notifyPendingApplicants(ctx, job, "Gig updated",
		fmt.Sprintf("'%s' was updated by the client — check the latest details.", job.Title))

	return job, nil
}

// CloseJob manually stops a job accepting applications without hiring
// anyone — distinct from a job going inactive because someone was hired.
// If escrow was already funded, the held amount is refunded back to the
// client's wallet balance since there's no counterparty left to pay it to.
func (u *jobUsecase) CloseJob(ctx context.Context, clientID, jobID string) (*domain.Job, error) {
	job, err := u.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	if job.ClientID != clientID {
		return nil, errors.New("unauthorized: only the job's creator can close it")
	}
	if job.Status != "open" && job.Status != "pending_funding" {
		return nil, fmt.Errorf("job can't be closed from status %q", job.Status)
	}

	if job.EscrowFunded && job.MusicianID == "" {
		wallet, err := u.walletRepo.GetOrCreate(ctx, clientID)
		if err == nil {
			wallet.Balance += job.EscrowAmount
			wallet.EscrowBalance -= job.EscrowAmount
			if err := u.walletRepo.Save(ctx, wallet); err == nil {
				_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
					UserID: clientID, Type: "escrow_release", Amount: job.EscrowAmount,
					Description: fmt.Sprintf("Escrow refunded — gig closed: %s", job.Title),
				})
			}
		}
	}

	job.Status = "closed"
	job.UpdatedAt = time.Now()
	if err := u.jobRepo.Update(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to close job: %w", err)
	}

	u.notifyPendingApplicants(ctx, job, "Gig closed",
		fmt.Sprintf("'%s' was closed by the client and is no longer accepting applications.", job.Title))

	return job, nil
}

func (u *jobUsecase) notifyPendingApplicants(ctx context.Context, job *domain.Job, title, message string) {
	apps, err := u.jobRepo.ListApplications(ctx, job.ID)
	if err != nil {
		return
	}
	for _, app := range apps {
		if app.Status == "pending" {
			u.notify(ctx, app.MusicianID, title, message)
		}
	}
}

func (u *jobUsecase) notify(ctx context.Context, userID, title, message string) {
	_ = u.notifRepo.Create(ctx, &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	})
}

// FundEscrow moves the job's budget from the client's wallet balance into
// escrow and only then makes the job visible/open to applicants — the
// "Escrow funded" badge shown on job cards is a guarantee, not decoration.
func (u *jobUsecase) FundEscrow(ctx context.Context, clientID, jobID string) (*domain.Job, error) {
	job, err := u.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	if job.ClientID != clientID {
		return nil, errors.New("unauthorized: only the job's creator can fund escrow")
	}
	if job.Status != "pending_funding" {
		return nil, errors.New("job is not awaiting escrow funding")
	}

	wallet, err := u.walletRepo.GetOrCreate(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to load wallet: %w", err)
	}
	if wallet.Balance < job.Budget {
		return nil, fmt.Errorf("insufficient wallet balance: need %.2f, have %.2f — top up your wallet first", job.Budget, wallet.Balance)
	}

	wallet.Balance -= job.Budget
	wallet.EscrowBalance += job.Budget
	if err := u.walletRepo.Save(ctx, wallet); err != nil {
		return nil, fmt.Errorf("failed to fund escrow: %w", err)
	}
	_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
		UserID: clientID, Type: "escrow_hold", Amount: job.Budget,
		Description: fmt.Sprintf("Escrow funded for gig: %s", job.Title),
	})

	job.EscrowFunded = true
	job.EscrowAmount = job.Budget
	job.Status = "open"
	job.UpdatedAt = time.Now()
	if err := u.jobRepo.Update(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to activate job: %w", err)
	}

	return job, nil
}

func (u *jobUsecase) GetJob(ctx context.Context, id string) (*domain.Job, error) {
	job, err := u.jobRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if count, err := u.jobRepo.CountApplications(ctx, job.ID); err == nil {
		job.ApplicationCount = count
	}
	job.Client = u.buildClientInfo(ctx, job.ClientID)
	if job.Client != nil {
		job.ClientRating = job.Client.Rating
		job.ClientReviewCount = job.Client.ReviewCount
	}
	return job, nil
}

func (u *jobUsecase) ListJobs(ctx context.Context, filter domain.JobFilter) ([]*domain.Job, error) {
	jobs, err := u.jobRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if filter.MaxApplications > 0 {
		filtered := jobs[:0]
		for _, job := range jobs {
			count, err := u.jobRepo.CountApplications(ctx, job.ID)
			if err != nil {
				return nil, err
			}
			if count <= filter.MaxApplications {
				filtered = append(filtered, job)
			}
		}
		jobs = filtered
	}

	u.sortJobs(ctx, jobs, filter)
	u.populateJobStats(ctx, jobs)
	return jobs, nil
}

func (u *jobUsecase) RecommendedJobs(ctx context.Context, musicianID string, limit int, extra domain.JobFilter) ([]*domain.Job, error) {
	musician, err := u.userRepo.GetByID(ctx, musicianID)
	if err != nil {
		return nil, fmt.Errorf("musician validation failed: %w", err)
	}
	if musician.Role != "musician" {
		return nil, errors.New("only musicians can receive job recommendations")
	}

	filter := extra
	filter.Status = "open"
	// Personalization only fills in gaps the caller didn't already narrow
	// down — an explicit search/filter from the job board always wins.
	if filter.Genre == "" && musician.MusicianProfile != nil && len(musician.MusicianProfile.Genres) > 0 {
		filter.Genre = musician.MusicianProfile.Genres[0]
	}
	if filter.Instrument == "" && musician.MusicianProfile != nil && len(musician.MusicianProfile.Instruments) > 0 {
		filter.Instrument = musician.MusicianProfile.Instruments[0]
	}
	if filter.Location == "" {
		filter.Location = musician.Location
	}
	filter.SortBy = "relevance"
	filter.MusicianID = musicianID

	jobs, err := u.ListJobs(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		fallback := domain.JobFilter{Status: "open", SortBy: "newest", Query: extra.Query}
		jobs, err = u.ListJobs(ctx, fallback)
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

func (u *jobUsecase) ApplyForJob(ctx context.Context, musicianID, jobID, proposal string, priceBid float64, portfolioItemIDs []string) (*domain.JobApplication, error) {
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
		JobID:          jobID,
		MusicianID:     musicianID,
		Proposal:       proposal,
		PriceBid:       priceBid,
		Status:         "pending",
		PortfolioItems: selectPortfolioItems(user.MusicianProfile, portfolioItemIDs),
		CreatedAt:      time.Now(),
	}

	if err := u.jobRepo.CreateApplication(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to submit application: %w", err)
	}

	applicantName := user.Name
	if user.MusicianProfile != nil && user.MusicianProfile.StageName != "" {
		applicantName = user.MusicianProfile.StageName
	}
	u.notifyAndEmail(ctx, job.ClientID, "New Application", fmt.Sprintf("%s applied to your gig '%s'.", applicantName, job.Title), "/dashboard/client")

	return app, nil
}

// selectPortfolioItems snapshots the musician's portfolio items matching
// the requested IDs, in the order they were selected — a nil/empty
// selection is a normal, valid application (attaching work samples is
// optional), so this never errors, it just returns nothing to attach.
func selectPortfolioItems(profile *domain.MusicianProfile, ids []string) []domain.PortfolioItem {
	if profile == nil || len(ids) == 0 {
		return nil
	}
	byID := make(map[string]domain.PortfolioItem, len(profile.Portfolio))
	for _, item := range profile.Portfolio {
		if item.ID != "" {
			byID[item.ID] = item
		}
	}
	selected := make([]domain.PortfolioItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			selected = append(selected, item)
		}
	}
	return selected
}

func (u *jobUsecase) ListJobApplications(ctx context.Context, jobID string) ([]*domain.JobApplication, error) {
	apps, err := u.jobRepo.ListApplications(ctx, jobID)
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		app.Applicant = u.buildApplicantSummary(ctx, app.MusicianID)
	}
	return apps, nil
}

// buildApplicantSummary is the at-a-glance context a client sees per
// applicant (rating, genres, instruments) — best-effort, a lookup failure
// just means that one application shows without the summary rather than
// failing the whole list.
func (u *jobUsecase) buildApplicantSummary(ctx context.Context, musicianID string) *domain.ApplicantSummary {
	musician, err := u.userRepo.GetByID(ctx, musicianID)
	if err != nil {
		return nil
	}
	summary := &domain.ApplicantSummary{
		Name:     musician.Name,
		Location: musician.Location,
	}
	if avg, n, err := u.averageRating(ctx, musicianID); err == nil {
		summary.Rating = avg
		summary.ReviewCount = n
	}
	if musician.MusicianProfile != nil {
		summary.Genres = musician.MusicianProfile.Genres
		summary.Instruments = musician.MusicianProfile.Instruments
	}
	return summary
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

func (u *jobUsecase) AcceptApplication(ctx context.Context, clientID, applicationID string) (*domain.Contract, error) {
	app, err := u.jobRepo.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	job, err := u.jobRepo.GetByID(ctx, app.JobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	if job.ClientID != clientID {
		return nil, errors.New("unauthorized: only the job creator can accept applications")
	}

	if job.Status != "open" {
		return nil, errors.New("job is no longer open")
	}

	// Update application status
	app.Status = "accepted"
	if err := u.jobRepo.UpdateApplication(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to update application status: %w", err)
	}

	// Update job status and set hired musician
	job.Status = "active"
	job.MusicianID = app.MusicianID
	job.UpdatedAt = time.Now()
	if err := u.jobRepo.Update(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
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
		return nil, fmt.Errorf("failed to create contract: %w", err)
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
	contractLink := "/contracts/" + contract.ID
	u.notifyAndEmail(ctx, job.ClientID, "Application Accepted", fmt.Sprintf("You have hired musician '%s' for gig '%s'. Price: $%.2f", app.MusicianID, job.Title, app.PriceBid), contractLink)
	u.notifyAndEmail(ctx, app.MusicianID, "Bid Accepted", fmt.Sprintf("Congratulations! Your proposal for gig '%s' was accepted. Price: $%.2f", job.Title, app.PriceBid), contractLink)

	return contract, nil
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
			count, err := u.jobRepo.CountApplications(ctx, job.ID)
			if err == nil {
				counts[job.ID] = count
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
		if containsFold(musician.MusicianProfile.Genres, job.Genre) {
			score += 3
		}
		if containsFold(musician.MusicianProfile.Instruments, job.Instrument) {
			score += 3
		}
	}
	if strings.EqualFold(job.Location, musician.Location) {
		score += 2
	}
	return score
}

func containsFold(values []string, target string) bool {
	if target == "" {
		return false
	}
	for _, v := range values {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}

func (u *jobUsecase) SaveJob(ctx context.Context, musicianID, jobID string) error {
	if _, err := u.jobRepo.GetByID(ctx, jobID); err != nil {
		return errors.New("job not found")
	}
	musician, err := u.userRepo.GetByID(ctx, musicianID)
	if err != nil {
		return err
	}
	if musician.Role != "musician" {
		return errors.New("only musicians can save jobs")
	}
	if musician.MusicianProfile == nil {
		musician.MusicianProfile = &domain.MusicianProfile{}
	}
	if containsFold(musician.MusicianProfile.SavedJobIDs, jobID) {
		return nil
	}
	musician.MusicianProfile.SavedJobIDs = append(musician.MusicianProfile.SavedJobIDs, jobID)
	return u.userRepo.Update(ctx, musician)
}

func (u *jobUsecase) UnsaveJob(ctx context.Context, musicianID, jobID string) error {
	musician, err := u.userRepo.GetByID(ctx, musicianID)
	if err != nil {
		return err
	}
	if musician.MusicianProfile == nil {
		return nil
	}
	kept := musician.MusicianProfile.SavedJobIDs[:0]
	for _, id := range musician.MusicianProfile.SavedJobIDs {
		if id != jobID {
			kept = append(kept, id)
		}
	}
	musician.MusicianProfile.SavedJobIDs = kept
	return u.userRepo.Update(ctx, musician)
}

func (u *jobUsecase) ListSavedJobs(ctx context.Context, musicianID string) ([]*domain.Job, error) {
	musician, err := u.userRepo.GetByID(ctx, musicianID)
	if err != nil {
		return nil, err
	}
	if musician.MusicianProfile == nil || len(musician.MusicianProfile.SavedJobIDs) == 0 {
		return []*domain.Job{}, nil
	}
	jobs := make([]*domain.Job, 0, len(musician.MusicianProfile.SavedJobIDs))
	for _, id := range musician.MusicianProfile.SavedJobIDs {
		job, err := u.jobRepo.GetByID(ctx, id)
		if err != nil {
			continue // job was deleted since being saved — skip it silently
		}
		jobs = append(jobs, job)
	}
	u.populateJobStats(ctx, jobs)
	return jobs, nil
}

// populateJobStats attaches read-only, query-time-only stats (proposal
// count, client rating) to each job in a list — used everywhere a job
// board card is rendered so it can show real numbers instead of nothing.
func (u *jobUsecase) populateJobStats(ctx context.Context, jobs []*domain.Job) {
	for _, job := range jobs {
		if count, err := u.jobRepo.CountApplications(ctx, job.ID); err == nil {
			job.ApplicationCount = count
		}
		if avg, n, err := u.averageRating(ctx, job.ClientID); err == nil {
			job.ClientRating = avg
			job.ClientReviewCount = n
		}
	}
}

func (u *jobUsecase) averageRating(ctx context.Context, userID string) (float64, int, error) {
	reviews, err := u.reviewRepo.ListByReviewee(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	if len(reviews) == 0 {
		return 0, 0, nil
	}
	sum := 0
	for _, rv := range reviews {
		sum += rv.Rating
	}
	return float64(sum) / float64(len(reviews)), len(reviews), nil
}

// buildClientInfo assembles the "About the client" panel shown on a job's
// detail view. Every field is derived from real jobs/contracts/reviews —
// best-effort throughout, since a stats lookup failure shouldn't 404 the
// job itself.
func (u *jobUsecase) buildClientInfo(ctx context.Context, clientID string) *domain.JobClientInfo {
	client, err := u.userRepo.GetByID(ctx, clientID)
	if err != nil {
		return nil
	}

	info := &domain.JobClientInfo{
		Name:        client.Name,
		Location:    client.Location,
		MemberSince: client.CreatedAt,
	}
	if client.ClientProfile != nil {
		info.CompanyName = client.ClientProfile.CompanyName
	}
	if avg, n, err := u.averageRating(ctx, clientID); err == nil {
		info.Rating = avg
		info.ReviewCount = n
	}

	if clientJobs, err := u.jobRepo.List(ctx, domain.JobFilter{ClientID: clientID}); err == nil {
		info.JobsPosted = len(clientJobs)
		hired := 0
		for _, j := range clientJobs {
			if j.Status == "open" {
				info.OpenJobs++
			}
			if j.MusicianID != "" {
				hired++
			}
		}
		if info.JobsPosted > 0 {
			info.HireRate = float64(hired) / float64(info.JobsPosted) * 100
		}
	}

	if contracts, err := u.contractRepo.ListForUser(ctx, clientID, "client"); err == nil {
		sort.SliceStable(contracts, func(i, j int) bool { return contracts[i].CreatedAt.After(contracts[j].CreatedAt) })
		for _, c := range contracts {
			if c.Status == "completed" {
				info.TotalSpent += c.Price
			}
		}
		const recentHiresLimit = 5
		for i, c := range contracts {
			if i >= recentHiresLimit {
				break
			}
			name := "Musician"
			if musician, err := u.userRepo.GetByID(ctx, c.MusicianID); err == nil && musician != nil {
				name = musician.Name
			}
			info.RecentHires = append(info.RecentHires, domain.JobClientHire{
				MusicianName: name,
				JobTitle:     c.Title,
				Status:       c.Status,
				Date:         c.CreatedAt,
			})
		}
	}

	return info
}

func (u *jobUsecase) notifyAndEmail(ctx context.Context, userID, title, message, link string) {
	notif := &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Link:      link,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)
	log.Printf("[EMAIL OUTBOX] To User %s: Subject: %s | Message: %s", userID, title, message)
}
