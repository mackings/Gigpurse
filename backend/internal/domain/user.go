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
	TermsAcceptedAt time.Time        `json:"terms_accepted_at,omitempty" bson:"terms_accepted_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" bson:"updated_at"`

	// Self-service presence/availability settings — both default false (a
	// user is neither hidden nor disabled until they opt in). "Disabled"
	// stays loggable-in on purpose: the point is a reversible "pause my
	// account" toggle, not a suspension, so the owner can always undo it.
	HidePresence bool `json:"hide_presence" bson:"hide_presence"`
	Disabled     bool `json:"disabled" bson:"disabled"`

	// Computed at query time only (never persisted — bson:"-" keeps them out
	// of Create/Update writes even if a caller round-trips a listed User).
	AverageRating      float64 `json:"average_rating,omitempty" bson:"-"`
	TotalReviews       int     `json:"total_reviews,omitempty" bson:"-"`
	CompletedContracts int     `json:"completed_contracts,omitempty" bson:"-"`
	TotalEarned        float64 `json:"total_earned,omitempty" bson:"-"`
	// Status is the presence status as seen by another user (never by the
	// account owner themselves) — one of "online", "offline", "disabled".
	// "hidden" is intentionally never exposed here: that's the whole point
	// of the HidePresence toggle, an observer sees "offline" instead.
	Status string `json:"status,omitempty" bson:"-"`
}

type MusicianProfile struct {
	StageName       string          `json:"stage_name" bson:"stage_name"`
	Instruments     []string        `json:"instruments" bson:"instruments"`
	Genres          []string        `json:"genres" bson:"genres"`
	ExperienceYears int             `json:"experience_years" bson:"experience_years"`
	PriceMin        float64         `json:"price_min,omitempty" bson:"price_min,omitempty"`
	PriceMax        float64         `json:"price_max,omitempty" bson:"price_max,omitempty"`
	Availability    []string        `json:"availability,omitempty" bson:"availability,omitempty"`
	SocialLinks     *SocialLinks    `json:"social_links,omitempty" bson:"social_links,omitempty"`
	IntroVideoURL   string          `json:"intro_video_url,omitempty" bson:"intro_video_url,omitempty"`
	Portfolio       []PortfolioItem `json:"portfolio,omitempty" bson:"portfolio,omitempty"`
	SavedJobIDs     []string        `json:"saved_job_ids,omitempty" bson:"saved_job_ids,omitempty"`
}

type SocialLinks struct {
	Instagram  string `json:"instagram,omitempty" bson:"instagram,omitempty"`
	Twitter    string `json:"twitter,omitempty" bson:"twitter,omitempty"`
	Facebook   string `json:"facebook,omitempty" bson:"facebook,omitempty"`
	YouTube    string `json:"youtube,omitempty" bson:"youtube,omitempty"`
	TikTok     string `json:"tiktok,omitempty" bson:"tiktok,omitempty"`
	Spotify    string `json:"spotify,omitempty" bson:"spotify,omitempty"`
	SoundCloud string `json:"soundcloud,omitempty" bson:"soundcloud,omitempty"`
	AppleMusic string `json:"apple_music,omitempty" bson:"apple_music,omitempty"`
}

type PortfolioItem struct {
	Title        string `json:"title" bson:"title"`
	Description  string `json:"description" bson:"description"`
	URL          string `json:"url" bson:"url"`
	MediaType    string `json:"media_type,omitempty" bson:"media_type,omitempty"` // "video", "audio", or "image"
	ExternalURL  string `json:"external_url,omitempty" bson:"external_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty" bson:"thumbnail_url,omitempty"`
	IsFeatured   bool   `json:"is_featured" bson:"is_featured"`
	Order        int    `json:"order" bson:"order"`
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
	SignUp(ctx context.Context, email, password, role, name string, acceptedTerms bool) (*User, error)
	Login(ctx context.Context, email, password string) (string, *User, error) // Returns JWT token and User
	ResendEmailVerification(ctx context.Context, email string) error
	VerifyEmail(ctx context.Context, email, code string) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	GetProfile(ctx context.Context, id string) (*User, error)
	UpdateProfile(ctx context.Context, id string, name, bio, location string, musProfile *MusicianProfile, cliProfile *ClientProfile) (*User, error)
	BrowseMusicians(ctx context.Context, filter MusicianFilter) ([]*User, error)
	UpdateAccountStatus(ctx context.Context, id string, hidePresence, disabled bool) (*User, error)
	GetUserStatus(ctx context.Context, id string) (string, error)
}
