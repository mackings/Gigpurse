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
	mux.HandleFunc("/jobs/save", JWTMiddleware(h.SaveJob))
	mux.HandleFunc("/jobs/unsave", JWTMiddleware(h.UnsaveJob))
	mux.HandleFunc("/jobs/saved", JWTMiddleware(h.ListSavedJobs))
	mux.HandleFunc("/jobs/fund", JWTMiddleware(h.FundEscrow))
	mux.HandleFunc("/jobs/close", JWTMiddleware(h.CloseJob))
}

func (h *JobHandler) HandleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListOrGetJob(w, r)
	case http.MethodPost:
		JWTMiddleware(h.PostJob)(w, r)
	case http.MethodPut:
		JWTMiddleware(h.UpdateJob)(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *JobHandler) ListOrGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		job, err := h.jobUsecase.GetJob(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusNotFound, "job_not_found", err.Error())
			return
		}
		respondSuccess(w, http.StatusOK, "job retrieved successfully", job)
		return
	}

	q := r.URL.Query()
	minB, _ := strconv.ParseFloat(q.Get("min_budget"), 64)
	maxB, _ := strconv.ParseFloat(q.Get("max_budget"), 64)
	maxApps, _ := strconv.Atoi(q.Get("max_applications"))

	filter := domain.JobFilter{
		Query:           q.Get("query"),
		Status:          q.Get("status"),
		Genre:           q.Get("genre"),
		Instrument:      q.Get("instrument"),
		Location:        q.Get("location"),
		MinBudget:       minB,
		MaxBudget:       maxB,
		SortBy:          q.Get("sort_by"),
		SortOrder:       q.Get("sort_order"),
		MaxApplications: maxApps,
		ClientID:        q.Get("client_id"),
	}

	jobs, err := h.jobUsecase.ListJobs(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "jobs_list_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "jobs retrieved successfully", jobs)
}

func (h *JobHandler) RecommendedJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can view recommendations")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	minB, _ := strconv.ParseFloat(q.Get("min_budget"), 64)
	maxB, _ := strconv.ParseFloat(q.Get("max_budget"), 64)
	extra := domain.JobFilter{
		Query:      q.Get("query"),
		Genre:      q.Get("genre"),
		Instrument: q.Get("instrument"),
		Location:   q.Get("location"),
		MinBudget:  minB,
		MaxBudget:  maxB,
	}
	jobs, err := h.jobUsecase.RecommendedJobs(r.Context(), userID, limit, extra)
	if err != nil {
		respondError(w, http.StatusBadRequest, "recommendations_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "recommended jobs retrieved successfully", jobs)
}

func (h *JobHandler) MyJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can view talent jobs")
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	jobs, err := h.jobUsecase.ListMusicianJobsByStatus(r.Context(), userID, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "musician_jobs_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "musician jobs retrieved successfully", jobs)
}

func (h *JobHandler) PostJob(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "client" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only clients can post jobs")
		return
	}

	var req struct {
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		Instrument      string   `json:"instrument"`
		Genre           string   `json:"genre"`
		Location        string   `json:"location"`
		Budget          float64  `json:"budget"`
		ExperienceLevel string   `json:"experience_level"`
		Duration        string   `json:"duration"`
		ProjectType     string   `json:"project_type"`
		Skills          []string `json:"skills"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	job, err := h.jobUsecase.PostJob(r.Context(), userID, domain.JobPostInput{
		Title:           req.Title,
		Description:     req.Description,
		Instrument:      req.Instrument,
		Genre:           req.Genre,
		Location:        req.Location,
		Budget:          req.Budget,
		ExperienceLevel: req.ExperienceLevel,
		Duration:        req.Duration,
		ProjectType:     req.ProjectType,
		Skills:          req.Skills,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "job_create_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusCreated, "job created successfully", job)
}

func (h *JobHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "client" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only clients can edit jobs")
		return
	}

	var req struct {
		JobID           string   `json:"job_id"`
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		Instrument      string   `json:"instrument"`
		Genre           string   `json:"genre"`
		Location        string   `json:"location"`
		Budget          float64  `json:"budget"`
		ExperienceLevel string   `json:"experience_level"`
		Duration        string   `json:"duration"`
		ProjectType     string   `json:"project_type"`
		Skills          []string `json:"skills"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	job, err := h.jobUsecase.UpdateJob(r.Context(), userID, req.JobID, domain.JobPostInput{
		Title:           req.Title,
		Description:     req.Description,
		Instrument:      req.Instrument,
		Genre:           req.Genre,
		Location:        req.Location,
		Budget:          req.Budget,
		ExperienceLevel: req.ExperienceLevel,
		Duration:        req.Duration,
		ProjectType:     req.ProjectType,
		Skills:          req.Skills,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "job_update_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "job updated successfully", job)
}

func (h *JobHandler) CloseJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "client" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only clients can close jobs")
		return
	}
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	job, err := h.jobUsecase.CloseJob(r.Context(), userID, req.JobID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "job_close_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "job closed successfully", job)
}

func (h *JobHandler) FundEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "client" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only clients can fund escrow")
		return
	}
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	job, err := h.jobUsecase.FundEscrow(r.Context(), userID, req.JobID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "fund_escrow_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "escrow funded — job is now live", job)
}

func (h *JobHandler) ApplyForJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can apply for jobs")
		return
	}

	var req struct {
		JobID            string   `json:"job_id"`
		Proposal         string   `json:"proposal"`
		PriceBid         float64  `json:"price_bid"`
		PortfolioItemIDs []string `json:"portfolio_item_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	app, err := h.jobUsecase.ApplyForJob(r.Context(), userID, req.JobID, req.Proposal, req.PriceBid, req.PortfolioItemIDs)
	if err != nil {
		respondError(w, http.StatusBadRequest, "job_application_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusCreated, "application submitted successfully", app)
}

func (h *JobHandler) HandleApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	jobID := r.URL.Query().Get("job_id")
	if jobID != "" {
		// Verify requesting user is the client who posted the job
		job, err := h.jobUsecase.GetJob(r.Context(), jobID)
		if err != nil {
			respondError(w, http.StatusNotFound, "job_not_found", "job not found")
			return
		}
		if job.ClientID != userID {
			respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only the job creator can view applications")
			return
		}

		apps, err := h.jobUsecase.ListJobApplications(r.Context(), jobID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "applications_list_failed", err.Error())
			return
		}

		respondSuccess(w, http.StatusOK, "applications retrieved successfully", apps)
		return
	}

	// If no jobID and user is musician, return their own applications
	if role == "musician" {
		apps, err := h.jobUsecase.ListApplicationsByMusician(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "applications_list_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusOK, "applications retrieved successfully", apps)
		return
	}

	respondError(w, http.StatusBadRequest, "missing_job_id", "job_id query parameter is required for clients")
}

func (h *JobHandler) AcceptApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "client" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only clients can accept applications")
		return
	}

	var req struct {
		ApplicationID string `json:"application_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	contract, err := h.jobUsecase.AcceptApplication(r.Context(), userID, req.ApplicationID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "application_accept_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "application accepted successfully", map[string]string{
		"message":     "application accepted successfully, job is now active",
		"contract_id": contract.ID,
	})
}

func (h *JobHandler) SaveJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can save jobs")
		return
	}
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.jobUsecase.SaveJob(r.Context(), userID, req.JobID); err != nil {
		respondError(w, http.StatusBadRequest, "save_job_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "job saved successfully", map[string]string{"job_id": req.JobID})
}

func (h *JobHandler) UnsaveJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can unsave jobs")
		return
	}
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.jobUsecase.UnsaveJob(r.Context(), userID, req.JobID); err != nil {
		respondError(w, http.StatusBadRequest, "unsave_job_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "job unsaved successfully", map[string]string{"job_id": req.JobID})
}

func (h *JobHandler) ListSavedJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can view saved jobs")
		return
	}
	jobs, err := h.jobUsecase.ListSavedJobs(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "saved_jobs_list_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "saved jobs retrieved successfully", jobs)
}
