package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gigpurse/internal/domain"
)

type JobHandler struct {
	jobUsecase domain.JobUsecase
}

func NewJobHandler(ju domain.JobUsecase) *JobHandler {
	return &JobHandler{
		jobUsecase: ju,
	}
}

func (h *JobHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/jobs", h.HandleJobs)
	mux.HandleFunc("/jobs/recommended", JWTMiddleware(h.RecommendedJobs))
	mux.HandleFunc("/jobs/mine", JWTMiddleware(h.MyJobs))
	mux.HandleFunc("/jobs/apply", JWTMiddleware(h.ApplyForJob))
	mux.HandleFunc("/jobs/applications", JWTMiddleware(h.HandleApplications))
	mux.HandleFunc("/jobs/applications/accept", JWTMiddleware(h.AcceptApplication))
}

func (h *JobHandler) HandleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListOrGetJob(w, r)
	case http.MethodPost:
		JWTMiddleware(h.PostJob)(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *JobHandler) ListOrGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		job, err := h.jobUsecase.GetJob(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
		return
	}

	q := r.URL.Query()
	minB, _ := strconv.ParseFloat(q.Get("min_budget"), 64)
	maxB, _ := strconv.ParseFloat(q.Get("max_budget"), 64)
	maxApps, _ := strconv.Atoi(q.Get("max_applications"))

	filter := domain.JobFilter{
		Status:          q.Get("status"),
		Genre:           q.Get("genre"),
		Instrument:      q.Get("instrument"),
		Location:        q.Get("location"),
		MinBudget:       minB,
		MaxBudget:       maxB,
		SortBy:          q.Get("sort_by"),
		SortOrder:       q.Get("sort_order"),
		MaxApplications: maxApps,
	}

	jobs, err := h.jobUsecase.ListJobs(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (h *JobHandler) RecommendedJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		http.Error(w, "unauthorized: only musicians can view recommendations", http.StatusForbidden)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	jobs, err := h.jobUsecase.RecommendedJobs(r.Context(), userID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (h *JobHandler) MyJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		http.Error(w, "unauthorized: only musicians can view talent jobs", http.StatusForbidden)
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	jobs, err := h.jobUsecase.ListMusicianJobsByStatus(r.Context(), userID, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (h *JobHandler) PostJob(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "client" {
		http.Error(w, "unauthorized: only clients can post jobs", http.StatusForbidden)
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Instrument  string  `json:"instrument"`
		Genre       string  `json:"genre"`
		Location    string  `json:"location"`
		Budget      float64 `json:"budget"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	job, err := h.jobUsecase.PostJob(r.Context(), userID, req.Title, req.Description, req.Instrument, req.Genre, req.Location, req.Budget)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func (h *JobHandler) ApplyForJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		http.Error(w, "unauthorized: only musicians can apply for jobs", http.StatusForbidden)
		return
	}

	var req struct {
		JobID    string  `json:"job_id"`
		Proposal string  `json:"proposal"`
		PriceBid float64 `json:"price_bid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	app, err := h.jobUsecase.ApplyForJob(r.Context(), userID, req.JobID, req.Proposal, req.PriceBid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

func (h *JobHandler) HandleApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	jobID := r.URL.Query().Get("job_id")
	if jobID != "" {
		// Verify requesting user is the client who posted the job
		job, err := h.jobUsecase.GetJob(r.Context(), jobID)
		if err != nil {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		if job.ClientID != userID {
			http.Error(w, "unauthorized: only the job creator can view applications", http.StatusForbidden)
			return
		}

		apps, err := h.jobUsecase.ListJobApplications(r.Context(), jobID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apps)
		return
	}

	// If no jobID and user is musician, return their own applications
	if role == "musician" {
		apps, err := h.jobUsecase.ListApplicationsByMusician(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apps)
		return
	}

	http.Error(w, "job_id query parameter is required for clients", http.StatusBadRequest)
}

func (h *JobHandler) AcceptApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "client" {
		http.Error(w, "unauthorized: only clients can accept applications", http.StatusForbidden)
		return
	}

	var req struct {
		ApplicationID string `json:"application_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.jobUsecase.AcceptApplication(r.Context(), userID, req.ApplicationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "application accepted successfully, job is now active",
	})
}
