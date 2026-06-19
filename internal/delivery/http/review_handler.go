package http

import (
	"encoding/json"
	"net/http"

	"gigpurse/internal/domain"
)

type ReviewHandler struct {
	reviewUsecase domain.ReviewUsecase
}

func NewReviewHandler(ru domain.ReviewUsecase) *ReviewHandler {
	return &ReviewHandler{
		reviewUsecase: ru,
	}
}

func (h *ReviewHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/reviews", h.HandleReviews)
	mux.HandleFunc("/reviews/average", h.GetAverageRating)
}

func (h *ReviewHandler) HandleReviews(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		JWTMiddleware(h.SubmitReview)(w, r)
	case http.MethodGet:
		h.ListReviews(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *ReviewHandler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req struct {
		JobID   string `json:"job_id"`
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	review, err := h.reviewUsecase.SubmitReview(r.Context(), userID, req.JobID, req.Rating, req.Comment)
	if err != nil {
		respondError(w, http.StatusBadRequest, "review_submit_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusCreated, "review submitted successfully", review)
}

func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "missing_user_id", "user_id query parameter is required")
		return
	}

	reviews, err := h.reviewUsecase.GetUserReviews(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "reviews_list_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "reviews retrieved successfully", reviews)
}

func (h *ReviewHandler) GetAverageRating(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "missing_user_id", "user_id query parameter is required")
		return
	}

	avg, err := h.reviewUsecase.GetAverageRating(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "average_rating_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "average rating retrieved successfully", map[string]interface{}{
		"user_id":        userID,
		"average_rating": avg,
	})
}
