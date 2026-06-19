package domain

import (
	"context"
	"time"
)

type User struct {
	ID              string           `json:"id" bson:"_id"`
	Email           string           `json:"email" bson:"email"`
	EmailVerified   bool             `json:"email_verified" bson:"email_verified"`
	PasswordHash    string           `json:"-" bson:"password_hash"`
	Role            string           `json:"role" bson:"role"` // "client" or "musician"
	Name            string           `json:"name" bson:"name"`
	Bio             string           `json:"bio" bson:"bio"`
	Location        string           `json:"location" bson:"location"`
	MusicianProfile *MusicianProfile `json:"musician_profile,omitempty" bson:"musician_profile,omitempty"`
	ClientProfile   *ClientProfile   `json:"client_profile,omitempty" bson:"client_profile,omitempty"`
	CreatedAt       time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" bson:"updated_at"`
}

type MusicianProfile struct {
	StageName       string          `json:"stage_name" bson:"stage_name"`
	Instrument      string          `json:"instrument" bson:"instrument"`
	Genre           string          `json:"genre" bson:"genre"`
	ExperienceYears int             `json:"experience_years" bson:"experience_years"`
	Portfolio       []PortfolioItem `json:"portfolio,omitempty" bson:"portfolio,omitempty"`
}

type PortfolioItem struct {
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	URL         string `json:"url" bson:"url"`
}

type ClientProfile struct {
	CompanyName string `json:"company_name" bson:"company_name"`
}

type MusicianFilter struct {
	Genre      string `json:"genre"`
	Instrument string `json:"instrument"`
	Location   string `json:"location"`
	MinExp     int    `json:"min_exp"`
	SortBy     string `json:"sort_by"`    // "experience", "rating", "newest"
	SortOrder  string `json:"sort_order"` // "asc" or "desc"
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	ListMusicians(ctx context.Context, filter MusicianFilter) ([]*User, error)
}

type UserUsecase interface {
	SignUp(ctx context.Context, email, password, role, name string) (*User, error)
	Login(ctx context.Context, email, password string) (string, *User, error) // Returns JWT token and User
	ResendEmailVerification(ctx context.Context, email string) error
	VerifyEmail(ctx context.Context, email, code string) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	GetProfile(ctx context.Context, id string) (*User, error)
	UpdateProfile(ctx context.Context, id string, name, bio, location string, musProfile *MusicianProfile, cliProfile *ClientProfile) (*User, error)
	BrowseMusicians(ctx context.Context, filter MusicianFilter) ([]*User, error)
}
