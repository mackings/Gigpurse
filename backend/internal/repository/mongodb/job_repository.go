package mongodb

import (
	"context"
	"errors"

	"gigpurse/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type jobRepository struct {
	jobColl *mongo.Collection
	appColl *mongo.Collection
}

func NewJobRepository(db *mongo.Database) domain.JobRepository {
	return &jobRepository{
		jobColl: db.Collection("jobs"),
		appColl: db.Collection("job_applications"),
	}
}

func (r *jobRepository) Create(ctx context.Context, job *domain.Job) error {
	if job.ID == "" {
		job.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.jobColl.InsertOne(ctx, job)
	return err
}

func (r *jobRepository) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	var job domain.Job
	err := r.jobColl.FindOne(ctx, bson.M{"_id": id}).Decode(&job)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("job not found")
		}
		return nil, err
	}
	return &job, nil
}

func (r *jobRepository) Update(ctx context.Context, job *domain.Job) error {
	_, err := r.jobColl.ReplaceOne(ctx, bson.M{"_id": job.ID}, job)
	return err
}

func (r *jobRepository) List(ctx context.Context, filter domain.JobFilter) ([]*domain.Job, error) {
	query := bson.M{}

	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.Genre != "" {
		query["genre"] = bson.M{"$regex": filter.Genre, "$options": "i"}
	}
	if filter.Instrument != "" {
		query["instrument"] = bson.M{"$regex": filter.Instrument, "$options": "i"}
	}
	if filter.Location != "" {
		query["location"] = bson.M{"$regex": filter.Location, "$options": "i"}
	}
	if filter.ClientID != "" {
		query["client_id"] = filter.ClientID
	}
	if filter.Query != "" {
		query["$or"] = []bson.M{
			{"title": bson.M{"$regex": filter.Query, "$options": "i"}},
			{"description": bson.M{"$regex": filter.Query, "$options": "i"}},
		}
	}

	budgetQuery := bson.M{}
	hasBudgetFilter := false
	if filter.MinBudget > 0 {
		budgetQuery["$gte"] = filter.MinBudget
		hasBudgetFilter = true
	}
	if filter.MaxBudget > 0 {
		budgetQuery["$lte"] = filter.MaxBudget
		hasBudgetFilter = true
	}
	if hasBudgetFilter {
		query["budget"] = budgetQuery
	}

	cursor, err := r.jobColl.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	jobs := []*domain.Job{}
	for cursor.Next(ctx) {
		var job domain.Job
		if err := cursor.Decode(&job); err != nil {
			return nil, err
		}
		jobs = append(jobs, &job)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *jobRepository) CreateApplication(ctx context.Context, app *domain.JobApplication) error {
	if app.ID == "" {
		app.ID = primitive.NewObjectID().Hex()
	}
	_, err := r.appColl.InsertOne(ctx, app)
	return err
}

func (r *jobRepository) GetApplicationByID(ctx context.Context, id string) (*domain.JobApplication, error) {
	var app domain.JobApplication
	err := r.appColl.FindOne(ctx, bson.M{"_id": id}).Decode(&app)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("application not found")
		}
		return nil, err
	}
	return &app, nil
}

func (r *jobRepository) UpdateApplication(ctx context.Context, app *domain.JobApplication) error {
	_, err := r.appColl.ReplaceOne(ctx, bson.M{"_id": app.ID}, app)
	return err
}

func (r *jobRepository) ListApplications(ctx context.Context, jobID string) ([]*domain.JobApplication, error) {
	cursor, err := r.appColl.Find(ctx, bson.M{"job_id": jobID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var apps []*domain.JobApplication
	for cursor.Next(ctx) {
		var app domain.JobApplication
		if err := cursor.Decode(&app); err != nil {
			return nil, err
		}
		apps = append(apps, &app)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return apps, nil
}

func (r *jobRepository) CountApplications(ctx context.Context, jobID string) (int, error) {
	count, err := r.appColl.CountDocuments(ctx, bson.M{"job_id": jobID})
	return int(count), err
}

func (r *jobRepository) ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*domain.JobApplication, error) {
	cursor, err := r.appColl.Find(ctx, bson.M{"musician_id": musicianID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var apps []*domain.JobApplication
	for cursor.Next(ctx) {
		var app domain.JobApplication
		if err := cursor.Decode(&app); err != nil {
			return nil, err
		}
		apps = append(apps, &app)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return apps, nil
}
