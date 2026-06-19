package domain

import (
	"context"
	"time"
)

type Review struct {
	ID         string    `json:"id" bson:"_id"`
	JobID      string    `json:"job_id" bson:"job_id"`
	ReviewerID string    `json:"reviewer_id" bson:"reviewer_id"`
	RevieweeID string    `json:"reviewee_id" bson:"reviewee_id"`
	Rating     int       `json:"rating" bson:"rating"` // 1 to 5
	Comment    string    `json:"comment" bson:"comment"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
}

type ReviewRepository interface {
	Create(ctx context.Context, review *Review) error
	ListByReviewee(ctx context.Context, revieweeID string) ([]*Review, error)
	GetByJobAndReviewer(ctx context.Context, jobID, reviewerID string) (*Review, error)
}

type ReviewUsecase interface {
	SubmitReview(ctx context.Context, reviewerID, jobID string, rating int, comment string) (*Review, error)
	GetUserReviews(ctx context.Context, userID string) ([]*Review, error)
	GetAverageRating(ctx context.Context, userID string) (float64, error)
}
