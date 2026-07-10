package domain

import (
	"context"
	"time"
)

type Job struct {
	ID          string    `json:"id" bson:"_id"`
	ClientID    string    `json:"client_id" bson:"client_id"`
	Title       string    `json:"title" bson:"title"`
	Description string    `json:"description" bson:"description"`
	Budget      float64   `json:"budget" bson:"budget"`
	Instrument  string    `json:"instrument" bson:"instrument"`
	Genre       string    `json:"genre" bson:"genre"`
	Location    string    `json:"location" bson:"location"`
	Status      string    `json:"status" bson:"status"` // "open", "pending", "active", "completed", "disputed"
	MusicianID  string    `json:"musician_id,omitempty" bson:"musician_id,omitempty"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

type JobApplication struct {
	ID         string    `json:"id" bson:"_id"`
	JobID      string    `json:"job_id" bson:"job_id"`
	MusicianID string    `json:"musician_id" bson:"musician_id"`
	Proposal   string    `json:"proposal" bson:"proposal"`
	PriceBid   float64   `json:"price_bid" bson:"price_bid"`
	Status     string    `json:"status" bson:"status"` // "pending", "accepted", "rejected"
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
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
	List(ctx context.Context, filter JobFilter) ([]*Job, error)

	CreateApplication(ctx context.Context, app *JobApplication) error
	GetApplicationByID(ctx context.Context, id string) (*JobApplication, error)
	UpdateApplication(ctx context.Context, app *JobApplication) error
	ListApplications(ctx context.Context, jobID string) ([]*JobApplication, error)
	ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*JobApplication, error)
}

type JobUsecase interface {
	PostJob(ctx context.Context, clientID, title, description, instrument, genre, location string, budget float64) (*Job, error)
	GetJob(ctx context.Context, id string) (*Job, error)
	ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error)
	RecommendedJobs(ctx context.Context, musicianID string, limit int, extra JobFilter) ([]*Job, error)

	ApplyForJob(ctx context.Context, musicianID, jobID, proposal string, priceBid float64) (*JobApplication, error)
	ListJobApplications(ctx context.Context, jobID string) ([]*JobApplication, error)
	ListApplicationsByMusician(ctx context.Context, musicianID string) ([]*JobApplication, error)
	ListMusicianJobsByStatus(ctx context.Context, musicianID, status string) ([]*Job, error)
	AcceptApplication(ctx context.Context, clientID, applicationID string) error

	SaveJob(ctx context.Context, musicianID, jobID string) error
	UnsaveJob(ctx context.Context, musicianID, jobID string) error
	ListSavedJobs(ctx context.Context, musicianID string) ([]*Job, error)
}
