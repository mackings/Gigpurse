package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type adminUsecase struct {
	db            *mongo.Database
	userRepo      domain.UserRepository
	jobRepo       domain.JobRepository
	contractRepo  domain.ContractRepository
	milestoneRepo domain.MilestoneRepository
	escrowRepo    domain.EscrowAgreementRepository
	jobUsecase    domain.JobUsecase
}

func NewAdminUsecase(
	db *mongo.Database,
	ur domain.UserRepository,
	jr domain.JobRepository,
	cr domain.ContractRepository,
	mr domain.MilestoneRepository,
	er domain.EscrowAgreementRepository,
	ju domain.JobUsecase,
) domain.AdminUsecase {
	return &adminUsecase{
		db:            db,
		userRepo:      ur,
		jobRepo:       jr,
		contractRepo:  cr,
		milestoneRepo: mr,
		escrowRepo:    er,
		jobUsecase:    ju,
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

	usersByRole, err := u.countByField(ctx, "users", "role")
	if err != nil {
		return nil, err
	}
	jobsByStatus, err := u.countByField(ctx, "jobs", "status")
	if err != nil {
		return nil, err
	}
	contractsByStatus, err := u.countByField(ctx, "contracts", "status")
	if err != nil {
		return nil, err
	}
	disputesByStatus, err := u.countByField(ctx, "disputes", "status")
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newUsers7d, err := u.db.Collection("users").CountDocuments(ctx, bson.M{"created_at": bson.M{"$gte": now.AddDate(0, 0, -7)}})
	if err != nil {
		return nil, err
	}
	newUsers30d, err := u.db.Collection("users").CountDocuments(ctx, bson.M{"created_at": bson.M{"$gte": now.AddDate(0, 0, -30)}})
	if err != nil {
		return nil, err
	}

	// Funded agreements only — status != PENDING means the client actually
	// paid, matching the same set FinalizeFund/FinalizeHire operate on.
	funded := bson.M{"status": bson.M{"$ne": "PENDING"}}
	totalRevenue, err := u.sumField(ctx, "escrow_agreements", funded, "platform_fee_naira")
	if err != nil {
		return nil, err
	}
	totalGMV, err := u.sumFields(ctx, "escrow_agreements", funded, "amount_naira", "platform_fee_naira")
	if err != nil {
		return nil, err
	}
	heldInEscrow, err := u.sumField(ctx, "escrow_agreements", bson.M{
		"status": "ONGOING", "payout_status": "NONE", "refund_status": "NONE",
	}, "amount_naira")
	if err != nil {
		return nil, err
	}

	revenueSeries, err := u.timeSeries(ctx, "escrow_agreements", funded, "platform_fee_naira")
	if err != nil {
		return nil, err
	}
	signupsSeries, err := u.timeSeries(ctx, "users", bson.M{}, "")
	if err != nil {
		return nil, err
	}

	return &domain.AdminAnalytics{
		TotalUsers:           totalUsers,
		TotalJobs:            totalJobs,
		TotalMessages:        totalMessages,
		TotalContracts:       totalContracts,
		TotalDisputes:        totalDisputes,
		UsersByRole:          usersByRole,
		NewUsersLast7Days:    newUsers7d,
		NewUsersLast30Days:   newUsers30d,
		JobsByStatus:         jobsByStatus,
		ContractsByStatus:    contractsByStatus,
		DisputesByStatus:     disputesByStatus,
		TotalPlatformRevenue: totalRevenue,
		TotalGMV:             totalGMV,
		TotalHeldInEscrow:    heldInEscrow,
		RevenueTimeSeries:    revenueSeries,
		SignupsTimeSeries:    signupsSeries,
	}, nil
}

// countByField groups every document in collection by field and returns a
// map of value -> count. A missing/empty field is skipped.
func (u *adminUsecase) countByField(ctx context.Context, collection, field string) (map[string]int64, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{field: bson.M{"$nin": bson.A{"", nil}}}},
		bson.M{"$group": bson.M{"_id": "$" + field, "count": bson.M{"$sum": 1}}},
	}
	cursor, err := u.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	out := map[string]int64{}
	var rows []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.Count
	}
	return out, nil
}

// sumField sums a single numeric field across documents matching filter.
func (u *adminUsecase) sumField(ctx context.Context, collection string, filter bson.M, field string) (float64, error) {
	return u.sumFields(ctx, collection, filter, field)
}

// sumFields sums the total of one or more numeric fields added together
// (e.g. amount_naira + platform_fee_naira for GMV) across matching documents.
func (u *adminUsecase) sumFields(ctx context.Context, collection string, filter bson.M, fields ...string) (float64, error) {
	var expr any
	if len(fields) == 1 {
		expr = "$" + fields[0]
	} else {
		add := bson.A{}
		for _, f := range fields {
			add = append(add, "$"+f)
		}
		expr = bson.M{"$add": add}
	}
	pipeline := bson.A{
		bson.M{"$match": filter},
		bson.M{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": expr}}},
	}
	cursor, err := u.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		Total float64 `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].Total, nil
}

// timeSeries buckets matching documents from the last 30 days by calendar
// day (created_at). valueField sums that field per day; an empty
// valueField counts documents instead (used for signups).
func (u *adminUsecase) timeSeries(ctx context.Context, collection string, filter bson.M, valueField string) ([]domain.TimeSeriesPoint, error) {
	since := time.Now().AddDate(0, 0, -30)
	match := bson.M{"created_at": bson.M{"$gte": since}}
	for k, v := range filter {
		match[k] = v
	}

	valueExpr := bson.M{"$sum": 1}
	if valueField != "" {
		valueExpr = bson.M{"$sum": "$" + valueField}
	}
	pipeline := bson.A{
		bson.M{"$match": match},
		bson.M{"$group": bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$created_at"}},
			"value": valueExpr,
		}},
		bson.M{"$sort": bson.M{"_id": 1}},
	}
	cursor, err := u.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rows []struct {
		ID    string  `bson:"_id"`
		Value float64 `bson:"value"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	points := make([]domain.TimeSeriesPoint, len(rows))
	for i, r := range rows {
		points[i] = domain.TimeSeriesPoint{Date: r.ID, Value: r.Value}
	}
	return points, nil
}

// userActivity is one collection's contribution to a user's engagement
// profile — the most recent event timestamp, how many events fell inside
// the requested window, and how many fell all-time (needed for the
// client-only tenure-normalized cadence figure).
type userActivity struct {
	Last         time.Time `bson:"last"`
	WindowCount  int64     `bson:"window_count"`
	AllTimeCount int64     `bson:"all_time_count"`
}

// lastAndCountByUser aggregates one collection, grouped by userField, into
// each user's most recent dateField timestamp plus window/all-time event
// counts. Multiple calls (one per event-source collection) get merged in
// Go by buildEngagement — simpler and easier to extend than one combined
// $unionWith pipeline, and fine at this data scale.
func (u *adminUsecase) lastAndCountByUser(ctx context.Context, collection, userField, dateField string, userIDs []string, since time.Time) (map[string]userActivity, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{userField: bson.M{"$in": userIDs}}},
		bson.M{"$group": bson.M{
			"_id":            "$" + userField,
			"last":           bson.M{"$max": "$" + dateField},
			"window_count":   bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$gte": bson.A{"$" + dateField, since}}, 1, 0}}},
			"all_time_count": bson.M{"$sum": 1},
		}},
	}
	cursor, err := u.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	out := map[string]userActivity{}
	// Flat fields, not an embedded/inline userActivity — bson:",inline" on
	// an embedded named type silently decoded to zero values in testing
	// here (mongo-driver quirk), where explicit flat fields decode fine.
	var rows []struct {
		ID           string    `bson:"_id"`
		Last         time.Time `bson:"last"`
		WindowCount  int64     `bson:"window_count"`
		AllTimeCount int64     `bson:"all_time_count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = userActivity{Last: r.Last, WindowCount: r.WindowCount, AllTimeCount: r.AllTimeCount}
	}
	return out, nil
}

// sumByUser is sumFields grouped by userField instead of a single total.
func (u *adminUsecase) sumByUser(ctx context.Context, collection, userField string, filter bson.M, fields ...string) (map[string]float64, error) {
	var expr any
	if len(fields) == 1 {
		expr = "$" + fields[0]
	} else {
		add := bson.A{}
		for _, f := range fields {
			add = append(add, "$"+f)
		}
		expr = bson.M{"$add": add}
	}
	pipeline := bson.A{
		bson.M{"$match": filter},
		bson.M{"$group": bson.M{"_id": "$" + userField, "total": bson.M{"$sum": expr}}},
	}
	cursor, err := u.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	out := map[string]float64{}
	var rows []struct {
		ID    string  `bson:"_id"`
		Total float64 `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.Total
	}
	return out, nil
}

// countByUser is countByField grouped by an explicit userField/filter
// instead of a free-form value field — used for gigs-completed counts.
func (u *adminUsecase) countByUser(ctx context.Context, collection, userField string, filter bson.M) (map[string]int64, error) {
	pipeline := bson.A{
		bson.M{"$match": filter},
		bson.M{"$group": bson.M{"_id": "$" + userField, "count": bson.M{"$sum": 1}}},
	}
	cursor, err := u.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	out := map[string]int64{}
	var rows []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.Count
	}
	return out, nil
}

func (u *adminUsecase) usersByRole(ctx context.Context, role string) ([]*domain.User, error) {
	cursor, err := u.db.Collection("users").Find(ctx, bson.M{"role": role})
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

// engagementSource is one collection contributing to "did this user do
// anything" — see ListTalentEngagement/ListClientEngagement for what counts.
type engagementSource struct {
	collection string
	userField  string
	dateField  string
}

// buildEngagement merges every engagementSource into a per-user last-active
// timestamp and window-scoped event count, then attaches a financial total
// and completed-gigs count from their own single sources. Also returns
// each user's all-time (not window-scoped) event count, which the
// client-only average-per-month figure needs and the talent path ignores.
func (u *adminUsecase) buildEngagement(
	ctx context.Context,
	users []*domain.User,
	windowDays int,
	sources []engagementSource,
	financialCollection, financialUserField string,
	financialFields []string,
	gigsCollection, gigsUserField string,
) ([]*domain.EngagementSummary, map[string]int64, error) {
	userIDs := make([]string, len(users))
	for i, us := range users {
		userIDs[i] = us.ID
	}
	since := time.Now().AddDate(0, 0, -windowDays)

	lastEngaged := map[string]time.Time{}
	windowCounts := map[string]int64{}
	allTimeCounts := map[string]int64{}
	for _, src := range sources {
		activity, err := u.lastAndCountByUser(ctx, src.collection, src.userField, src.dateField, userIDs, since)
		if err != nil {
			return nil, nil, err
		}
		for id, a := range activity {
			if a.Last.After(lastEngaged[id]) {
				lastEngaged[id] = a.Last
			}
			windowCounts[id] += a.WindowCount
			allTimeCounts[id] += a.AllTimeCount
		}
	}

	// Funded agreements only — matches the same set the admin overview's
	// revenue/GMV figures use (see GetAnalytics).
	financial, err := u.sumByUser(ctx, financialCollection, financialUserField, bson.M{"status": bson.M{"$ne": "PENDING"}}, financialFields...)
	if err != nil {
		return nil, nil, err
	}
	gigs, err := u.countByUser(ctx, gigsCollection, gigsUserField, bson.M{"status": "completed"})
	if err != nil {
		return nil, nil, err
	}

	summaries := make([]*domain.EngagementSummary, len(users))
	for i, us := range users {
		s := &domain.EngagementSummary{
			UserID:          us.ID,
			Name:            us.Name,
			Email:           us.Email,
			JoinedAt:        us.CreatedAt,
			EngagementCount: windowCounts[us.ID],
			GigsCount:       gigs[us.ID],
			FinancialTotal:  financial[us.ID],
		}
		if last, ok := lastEngaged[us.ID]; ok && !last.IsZero() {
			lastCopy := last
			s.LastEngagedAt = &lastCopy
		}
		summaries[i] = s
	}
	return summaries, allTimeCounts, nil
}

func (u *adminUsecase) ListTalentEngagement(ctx context.Context, windowDays int) ([]*domain.EngagementSummary, error) {
	musicians, err := u.usersByRole(ctx, "musician")
	if err != nil {
		return nil, err
	}
	summaries, _, err := u.buildEngagement(ctx, musicians, windowDays,
		[]engagementSource{
			{"job_applications", "musician_id", "created_at"},
			{"contracts", "musician_id", "created_at"},
			{"direct_hire_requests", "musician_id", "created_at"},
			{"chat_messages", "sender_id", "timestamp"},
		},
		"escrow_agreements", "counterparty_user_id", []string{"amount_naira"},
		"contracts", "musician_id",
	)
	return summaries, err
}

func (u *adminUsecase) ListClientEngagement(ctx context.Context, windowDays int) ([]*domain.EngagementSummary, error) {
	clients, err := u.usersByRole(ctx, "client")
	if err != nil {
		return nil, err
	}
	summaries, allTimeCounts, err := u.buildEngagement(ctx, clients, windowDays,
		[]engagementSource{
			{"jobs", "client_id", "created_at"},
			{"contracts", "client_id", "created_at"},
			{"direct_hire_requests", "client_id", "created_at"},
			{"chat_messages", "sender_id", "timestamp"},
		},
		"escrow_agreements", "initiator_user_id", []string{"amount_naira", "platform_fee_naira"},
		"contracts", "client_id",
	)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for _, s := range summaries {
		months := now.Sub(s.JoinedAt).Hours() / 24 / 30
		if months < 1 {
			months = 1 // a brand-new signup's cadence shouldn't get inflated by dividing by a fraction of a month
		}
		s.AvgEngagementPerMonth = float64(allTimeCounts[s.UserID]) / months
	}
	return summaries, nil
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

func (u *adminUsecase) GetJobDetail(ctx context.Context, jobID string) (*domain.AdminJobDetail, error) {
	job, err := u.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	apps, err := u.jobUsecase.ListJobApplications(ctx, jobID)
	if err != nil {
		return nil, err
	}
	detail := &domain.AdminJobDetail{Job: job, Applications: apps}

	// No contract yet is the common, non-error case (job still open, or
	// nobody's been hired) — every other lookup below is best-effort once
	// we know a contract actually exists.
	contract, err := u.contractRepo.GetByJobID(ctx, jobID)
	if err != nil || contract == nil {
		return detail, nil
	}
	detail.Contract = contract

	milestones, err := u.milestoneRepo.ListByContract(ctx, contract.ID)
	if err != nil {
		return detail, nil
	}
	for _, m := range milestones {
		summary := &domain.AdminMilestoneSummary{
			ID: m.ID, Title: m.Title, Amount: m.Amount, Status: m.Status,
			DueDate: m.DueDate, CreatedAt: m.CreatedAt,
		}
		if m.EscrowReference != "" {
			if agreement, err := u.escrowRepo.GetByReference(ctx, m.EscrowReference); err == nil {
				summary.EscrowStatus = agreement.Status
				summary.PayoutStatus = agreement.PayoutStatus
				summary.RefundStatus = agreement.RefundStatus
				summary.PlatformFee = agreement.PlatformFeeNaira
			}
		}
		detail.Milestones = append(detail.Milestones, summary)
	}
	return detail, nil
}

func (u *adminUsecase) DeleteJobListing(ctx context.Context, jobID string) error {
	_, err := u.db.Collection("jobs").DeleteOne(ctx, bson.M{"_id": jobID})
	return err
}

func (u *adminUsecase) InviteModerator(ctx context.Context, email, name string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	if existing, err := u.userRepo.GetByEmail(ctx, email); err == nil && existing != nil {
		return nil, fmt.Errorf("a user with this email already exists (role: %s)", existing.Role)
	}
	user := &domain.User{
		Email:     email,
		Role:      "moderator",
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to invite moderator: %w", err)
	}
	return user, nil
}

func (u *adminUsecase) ListModerators(ctx context.Context) ([]*domain.User, error) {
	cursor, err := u.db.Collection("users").Find(ctx, bson.M{"role": bson.M{"$in": bson.A{"moderator", "revoked_moderator"}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var moderators []*domain.User
	for cursor.Next(ctx) {
		var user domain.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		moderators = append(moderators, &user)
	}
	return moderators, nil
}

func (u *adminUsecase) SetModeratorStatus(ctx context.Context, userID string, active bool) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("moderator not found: %w", err)
	}
	if user.Role != "moderator" && user.Role != "revoked_moderator" {
		return errors.New("this user isn't a moderator")
	}
	if active {
		user.Role = "moderator"
	} else {
		user.Role = "revoked_moderator"
	}
	user.UpdatedAt = time.Now()
	return u.userRepo.Update(ctx, user)
}
