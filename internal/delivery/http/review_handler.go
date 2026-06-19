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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ReviewHandler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		JobID   string `json:"job_id"`
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	review, err := h.reviewUsecase.SubmitReview(r.Context(), userID, req.JobID, req.Rating, req.Comment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(review)
}

func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	reviews, err := h.reviewUsecase.GetUserReviews(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

func (h *ReviewHandler) GetAverageRating(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	avg, err := h.reviewUsecase.GetAverageRating(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":        userID,
		"average_rating": avg,
	})
}
