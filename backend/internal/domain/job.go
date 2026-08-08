package domain

import (
	"context"
	"time"
)

type Job struct {
	ID          string  `json:"id" bson:"_id"`
	ClientID    string  `json:"client_id" bson:"client_id"`
	Title       string  `json:"title" bson:"title"`
	Description string  `json:"description" bson:"description"`
	Budget      float64 `json:"budget" bson:"budget"`
	Instrument  string  `json:"instrument" bson:"instrument"`
	Genre       string  `json:"genre" bson:"genre"`
	Location    string  `json:"location" bson:"location"`
	Status      string  `json:"status" bson:"status"` // "pending_funding", "open", "pending_hire_funding", "active", "completed", "disputed", "closed"
	MusicianID  string  `json:"musician_id,omitempty" bson:"musician_id,omitempty"`

	ExperienceLevel string   `json:"experience_level,omitempty" bson:"experience_level,omitempty"` // "entry", "intermediate", "expert"
	Duration        string   `json:"duration,omitempty" bson:"duration,omitempty"`                 // "less_than_1_week", "1_to_2_weeks", "1_to_4_weeks", "1_to_3_months", "3_plus_months"
	ProjectType     string   `json:"project_type,omitempty" bson:"project_type,omitempty"`         // "one_time", "ongoing"
	Skills          []string `json:"skills,omitempty" bson:"skills,omitempty"`

	// Fixed-price escrow via PayPetal TrustCore. Money moves at hire time,
	// not post time — TrustCore requires both parties (initiator +
	// counterparty) to create an agreement, and no musician is known yet
	// when a job is posted. EscrowFunded/EscrowAmount stay false/zero
	// through "open" and only become true once InitiateHire's payment is
	// confirmed (see FinalizeHire); EscrowReference points at the
	// EscrowAgreement backing them.
	EscrowFunded    bool    `json:"escrow_funded" bson:"escrow_funded"`
	EscrowAmount    float64 `json:"escrow_amount,omitempty" bson:"escrow_amount,omitempty"`
	EscrowReference string  `json:"-" bson:"escrow_reference,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`

	// Computed at query time only (never persisted — bson:"-" keeps them out
	// of Create/Update writes even if a caller round-trips a listed Job).
	ApplicationCount  int            `json:"application_count,omitempty" bson:"-"`
	ClientRating      float64        `json:"client_rating,omitempty" bson:"-"`
	ClientReviewCount int            `json:"client_review_count,omitempty" bson:"-"`
	Client            *JobClientInfo `json:"client,omitempty" bson:"-"` // populated only on a single-job fetch
}

// JobClientInfo is the "About the client" panel shown on a job's detail
// view — every field is derived from real jobs/contracts/reviews data, no
// simulated stats (no Connects/bid-range/"last viewed" style filler).
type JobClientInfo struct {
	Name        string          `json:"name"`
	CompanyName string          `json:"company_name,omitempty"`
	Location    string          `json:"location,omitempty"`
	MemberSince time.Time       `json:"member_since"`
	Rating      float64         `json:"rating"`
	ReviewCount int             `json:"review_count"`
	JobsPosted  int             `json:"jobs_posted"`
	OpenJobs    int             `json:"open_jobs"`
	HireRate    float64         `json:"hire_rate"`   // % of posted jobs that resulted in a hire
	TotalSpent  float64         `json:"total_spent"` // sum of completed-contract prices
	RecentHires []JobClientHire `json:"recent_hires,omitempty"`
}

type JobClientHire struct {
	MusicianName string    `json:"musician_name"`
	JobTitle     string    `json:"job_title"`
	Status       string    `json:"status"`
	Date         time.Time `json:"date"`
}

type JobApplication struct {
	ID         string  `json:"id" bson:"_id"`
	JobID      string  `json:"job_id" bson:"job_id"`
	MusicianID string  `json:"musician_id" bson:"musician_id"`
	Proposal   string  `json:"proposal" bson:"proposal"`
	PriceBid   float64 `json:"price_bid" bson:"price_bid"`
	Status     string  `json:"status" bson:"status"` // "pending", "shortlisted", "accepted_pending_payment", "accepted", "rejected"
	// Snapshotted at application time (not a live reference) so the record
	// stays accurate even if the musician later edits/reorders/deletes
	// portfolio items — matches how job-detail "client info" is a snapshot,
	// not a live join.
	PortfolioItems []PortfolioItem `json:"portfolio_items,omitempty" bson:"portfolio_items,omitempty"`
	CreatedAt      time.Time       `json:"created_at" bson:"created_at"`

	// Computed at query time only (never persisted) — populated for the
	// client reviewing applicants, never needed from the musician's own side.
	Applicant *ApplicantSummary `json:"applicant,omitempty" bson:"-"`
}

// ApplicantSummary is the at-a-glance context a client sees when reviewing
// who applied to their job — real rating/genre data, not filler.
type ApplicantSummary struct {
	Name        string   `json:"name"`
	Location    string   `json:"location,omitempty"`
	Rating      float64  `json:"rating"`
	ReviewCount int      `json:"review_count"`
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

// JobPostInput bundles a job posting's fields — kept as a struct rather
// than growing PostJob's positional parameter list past readability.
type JobPostInput struct {
	Title           string
	Description     string
	Instrument      string
	Genre           string
	Location        string
	Budget          float64
	ExperienceLevel string
	Duration        string
	ProjectType     string
	Skills          []string
}

type JobFilter struct {
	Query           string  `json:"query"` // free-text search across title + description
	Status          string  `json:"status"`
	Genre           string  `json:"genre"`
	Instrument      string  `json:"instrument"`
	Location        string  `json:"location"`
	MinBudget       float64 `json:"min_budget"`
	MaxBudget       float64 `json:"max_budget"`
	SortBy          string  `json:"sort_by"` // "newest", "budget", "applications", "relevance"
	SortOrder       string  `json:"sort_order"`
	MaxApplications int     `json:"max_applications"`
	MusicianID      string  `json:"musician_id,omitempty"`
	ClientID        string  `json:"client_id,omitempty"`
}

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id string) (*Job, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter JobFilter) ([]*Job, error)

	CreateApplication(ctx context.Context, app *JobApplication) error
	GetApplicationByID(ctx context.Context, id string) (*JobApplication, error)
	UpdateApplication(ctx context.Context, app *JobApplication) error
	ListApplications(ctx context.Context, jobID string) ([]*JobApplication, error)
	ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*JobApplication, error)
	CountApplications(ctx context.Context, jobID string) (int, error)
}

type JobUsecase interface {
	PostJob(ctx context.Context, clientID string, input JobPostInput) (*Job, error)
	UpdateJob(ctx context.Context, clientID, jobID string, input JobPostInput) (*Job, error)
	CloseJob(ctx context.Context, clientID, jobID string) (*Job, error)
	DeleteJob(ctx context.Context, clientID, jobID string) error
	RequestHireRefund(ctx context.Context, clientID, referenceID string) error
	FundEscrow(ctx context.Context, clientID, jobID string) (*Job, error)
	GetJob(ctx context.Context, id string) (*Job, error)
	ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error)
	RecommendedJobs(ctx context.Context, musicianID string, limit int, extra JobFilter) ([]*Job, error)

	ApplyForJob(ctx context.Context, musicianID, jobID, proposal string, priceBid float64, portfolioItemIDs []string) (*JobApplication, error)
	ListJobApplications(ctx context.Context, jobID string) ([]*JobApplication, error)
	ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*JobApplication, error)
	ListMusicianJobsByStatus(ctx context.Context, musicianID, status string) ([]*Job, error)
	// ShortlistApplication/UnshortlistApplication toggle a lightweight,
	// non-committing "still considering" flag a client can set on a pending
	// applicant — purely organizational (doesn't touch payment); a
	// shortlisted application can still be accepted (hired) or reverted
	// back to pending at any time before the job is filled.
	ShortlistApplication(ctx context.Context, clientID, applicationID string) (*JobApplication, error)
	UnshortlistApplication(ctx context.Context, clientID, applicationID string) (*JobApplication, error)

	// InitiateHire replaces the old synchronous AcceptApplication — TrustCore
	// requires the client to actually pay via a hosted checkout before a
	// Contract exists, so accepting an application now returns a payment URL
	// instead of a Contract. FinalizeHire (called by the webhook, or by a
	// frontend poll after the client returns from checkout) does what
	// AcceptApplication used to do once payment is confirmed.
	InitiateHire(ctx context.Context, clientID, applicationID string) (paymentURL, reference string, err error)
	FinalizeHire(ctx context.Context, reference string) (*Contract, error)
	// StartAbandonedHireSweep runs in the background for the lifetime of
	// ctx, reverting a job stuck in "pending_hire_funding" back to "open"
	// if the client never completed checkout. Mirrors
	// MilestoneUsecase.StartReminderScanner's shape.
	StartAbandonedHireSweep(ctx context.Context, checkInterval time.Duration)

	SaveJob(ctx context.Context, musicianID, jobID string) error
	UnsaveJob(ctx context.Context, musicianID, jobID string) error
	ListSavedJobs(ctx context.Context, musicianID string) ([]*Job, error)
}
