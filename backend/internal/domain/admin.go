package domain

import (
	"context"
	"time"
)

// AdminMilestoneSummary is a purpose-built view for the admin job detail
// page — pulls in the escrow status a milestone's own EscrowReference
// points at (that field is json:"-" on Milestone itself, deliberately never
// exposed elsewhere) so an admin can see payment state without any other
// endpoint leaking PayPetal-internal references.
type AdminMilestoneSummary struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Amount       float64    `json:"amount"`
	Status       string     `json:"status"`
	DueDate      *time.Time `json:"due_date,omitempty"`
	EscrowStatus string     `json:"escrow_status,omitempty"`
	PayoutStatus string     `json:"payout_status,omitempty"`
	RefundStatus string     `json:"refund_status,omitempty"`
	PlatformFee  float64    `json:"platform_fee,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AdminJobDetail is everything an admin needs to see "what's happened" on
// one job at a glance — the job itself, who applied, and (once hired) the
// resulting contract and its milestone/escrow history.
type AdminJobDetail struct {
	Job          *Job                     `json:"job"`
	Applications []*JobApplication        `json:"applications"`
	Contract     *Contract                `json:"contract,omitempty"`
	Milestones   []*AdminMilestoneSummary `json:"milestones,omitempty"`
}

// TimeSeriesPoint is one day's value in a trend chart — Date is "2006-01-02".
type TimeSeriesPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type AdminAnalytics struct {
	TotalUsers     int64 `json:"total_users"`
	TotalJobs      int64 `json:"total_jobs"`
	TotalMessages  int64 `json:"total_messages"`
	TotalContracts int64 `json:"total_contracts"`
	TotalDisputes  int64 `json:"total_disputes"`

	UsersByRole        map[string]int64 `json:"users_by_role"`
	NewUsersLast7Days  int64            `json:"new_users_last_7_days"`
	NewUsersLast30Days int64            `json:"new_users_last_30_days"`

	JobsByStatus      map[string]int64 `json:"jobs_by_status"`
	ContractsByStatus map[string]int64 `json:"contracts_by_status"`
	DisputesByStatus  map[string]int64 `json:"disputes_by_status"`

	// Money figures are derived from EscrowAgreement — the one ledger of
	// what actually moved through PayPetal, not what jobs/milestones merely
	// list as their agreed price. See usecase/pricing.go for the split.
	TotalPlatformRevenue float64 `json:"total_platform_revenue"` // sum of PlatformFeeNaira on every funded agreement
	TotalGMV             float64 `json:"total_gmv"`              // sum of what clients actually paid (amount + fee)
	TotalHeldInEscrow    float64 `json:"total_held_in_escrow"`   // funded, not yet released or refunded

	RevenueTimeSeries []TimeSeriesPoint `json:"revenue_time_series"` // last 30 days
	SignupsTimeSeries []TimeSeriesPoint `json:"signups_time_series"` // last 30 days
}

// EngagementSummary is one user's activity/value profile for the admin
// Talent and Clients tabs — built to support churn-risk spotting (has this
// person gone quiet?) and top-performer ranking (financial/gig volume) in
// one row. See usecase/admin_usecase.go's ListTalentEngagement /
// ListClientEngagement for exactly what counts as an "engagement".
type EngagementSummary struct {
	UserID          string     `json:"user_id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	JoinedAt        time.Time  `json:"joined_at"`
	LastEngagedAt   *time.Time `json:"last_engaged_at,omitempty"`
	EngagementCount int64      `json:"engagement_count"` // within the requested window
	// AvgEngagementPerMonth is only meaningful (and only populated) for
	// clients — total engagement events, all-time, divided by months since
	// signup, so tenure doesn't skew the comparison between clients.
	AvgEngagementPerMonth float64 `json:"avg_engagement_per_month,omitempty"`
	GigsCount             int64   `json:"gigs_count"`
	// FinancialTotal is total earned for talent, total spent (GMV
	// generated) for clients.
	FinancialTotal float64 `json:"financial_total"`
}

type AdminUsecase interface {
	GetAnalytics(ctx context.Context) (*AdminAnalytics, error)
	ListAllUsers(ctx context.Context) ([]*User, error)
	ListAllJobs(ctx context.Context) ([]*Job, error)
	GetJobDetail(ctx context.Context, jobID string) (*AdminJobDetail, error)
	DeleteJobListing(ctx context.Context, jobID string) error

	// ListTalentEngagement / ListClientEngagement power the admin Talent and
	// Clients tabs — windowDays scopes EngagementCount (e.g. 30 = "active in
	// the last 30 days"); every other field is all-time.
	ListTalentEngagement(ctx context.Context, windowDays int) ([]*EngagementSummary, error)
	ListClientEngagement(ctx context.Context, windowDays int) ([]*EngagementSummary, error)

	// InviteModerator is the only way a moderator account comes into
	// existence — see the security note on RequestModeratorLogin in
	// usecase/user_usecase.go for why self-service creation was removed.
	InviteModerator(ctx context.Context, email, name string) (*User, error)
	ListModerators(ctx context.Context) ([]*User, error)
	// SetModeratorStatus flips a moderator's role between "moderator" and
	// "revoked_moderator" — the latter fails every existing admin/moderator
	// role check with no per-site changes needed. Blocks new logins
	// immediately; doesn't invalidate an already-issued JWT (see
	// middleware.go — no route does a live per-request DB check today).
	SetModeratorStatus(ctx context.Context, userID string, active bool) error
}
