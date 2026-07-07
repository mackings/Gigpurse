package domain

import (
	"context"
	"time"
)

type Contract struct {
	ID          string    `json:"id" bson:"_id"`
	JobID       string    `json:"job_id,omitempty" bson:"job_id,omitempty"`
	ClientID    string    `json:"client_id" bson:"client_id"`
	MusicianID  string    `json:"musician_id" bson:"musician_id"`
	Title       string    `json:"title,omitempty" bson:"title,omitempty"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	Price       float64   `json:"price" bson:"price"`
	Source      string    `json:"source" bson:"source"` // "job" or "direct_hire"
	Status      string    `json:"status" bson:"status"` // "active", "completed", "cancelled"
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// NegotiationEntry records one offer in a DirectHireRequest's back-and-forth
// — every propose/counter appends one, giving both parties a visible history
// of who offered what.
type NegotiationEntry struct {
	ProposedBy  string     `json:"proposed_by" bson:"proposed_by"`
	Price       float64    `json:"price" bson:"price"`
	Description string     `json:"description,omitempty" bson:"description,omitempty"`
	Location    string     `json:"location,omitempty" bson:"location,omitempty"`
	EventDate   *time.Time `json:"event_date,omitempty" bson:"event_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
}

type DirectHireRequest struct {
	ID          string             `json:"id" bson:"_id"`
	ClientID    string             `json:"client_id" bson:"client_id"`
	MusicianID  string             `json:"musician_id" bson:"musician_id"`
	Title       string             `json:"title" bson:"title"`
	Description string             `json:"description" bson:"description"`
	Location    string             `json:"location,omitempty" bson:"location,omitempty"`
	EventDate   *time.Time         `json:"event_date,omitempty" bson:"event_date,omitempty"`
	Price       float64            `json:"price" bson:"price"`
	// ProposedBy is whoever made the current (most recent) offer — the
	// other party is the one who can accept/decline/counter it.
	ProposedBy string             `json:"proposed_by" bson:"proposed_by"`
	History    []NegotiationEntry `json:"history,omitempty" bson:"history,omitempty"`
	Status     string             `json:"status" bson:"status"` // "pending", "accepted", "declined", "cancelled"
	ContractID string             `json:"contract_id,omitempty" bson:"contract_id,omitempty"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at"`
}

type ContractRepository interface {
	Create(ctx context.Context, contract *Contract) error
	GetByID(ctx context.Context, id string) (*Contract, error)
	GetByJobID(ctx context.Context, jobID string) (*Contract, error)
	Update(ctx context.Context, contract *Contract) error
	ListForUser(ctx context.Context, userID, role string) ([]*Contract, error)

	CreateDirectHireRequest(ctx context.Context, req *DirectHireRequest) error
	GetDirectHireRequestByID(ctx context.Context, id string) (*DirectHireRequest, error)
	UpdateDirectHireRequest(ctx context.Context, req *DirectHireRequest) error
	ListDirectHireRequestsForUser(ctx context.Context, userID, role, status string) ([]*DirectHireRequest, error)
}

type DirectHireTerms struct {
	Title       string
	Description string
	Location    string
	EventDate   *time.Time
	Price       float64
}

type ContractUsecase interface {
	GetContract(ctx context.Context, requesterID, requesterRole, id string) (*Contract, error)
	ListUserContracts(ctx context.Context, userID, role string) ([]*Contract, error)
	CompleteContract(ctx context.Context, clientID, contractID string) error
	CreateDirectHireRequest(ctx context.Context, clientID, musicianID string, terms DirectHireTerms) (*DirectHireRequest, error)
	ListDirectHireRequests(ctx context.Context, userID, role, status string) ([]*DirectHireRequest, error)
	GetDirectHireRequest(ctx context.Context, userID, requestID string) (*DirectHireRequest, error)
	RespondToDirectHireRequest(ctx context.Context, userID, requestID, decision string) (*DirectHireRequest, error)
	CounterDirectHireRequest(ctx context.Context, userID, requestID string, terms DirectHireTerms) (*DirectHireRequest, error)
}
