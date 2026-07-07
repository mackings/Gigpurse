package http

import (
	"encoding/json"
	"net/http"

	"gigpurse/internal/domain"
)

type AdminHandler struct {
	adminUsecase domain.AdminUsecase
}

func NewAdminHandler(au domain.AdminUsecase) *AdminHandler {
	return &AdminHandler{
		adminUsecase: au,
	}
}

func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin/analytics", JWTMiddleware(h.GetAnalytics))
	mux.HandleFunc("/admin/users", JWTMiddleware(h.ListUsers))
	mux.HandleFunc("/admin/jobs", JWTMiddleware(h.HandleJobs))
}

func (h *AdminHandler) checkAdmin(w http.ResponseWriter, r *http.Request) bool {
	_, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "admin" {
		respondError(w, http.StatusForbidden, "admin_required", "forbidden: admin role required")
		return false
	}
	return true
}

func (h *AdminHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.checkAdmin(w, r) {
		return
	}

	analytics, err := h.adminUsecase.GetAnalytics(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "analytics_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "analytics retrieved successfully", analytics)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.checkAdmin(w, r) {
		return
	}

	users, err := h.adminUsecase.ListAllUsers(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "users_list_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "users retrieved successfully", users)
}

func (h *AdminHandler) HandleJobs(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdmin(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.ListJobs(w, r)
	case http.MethodDelete:
		h.DeleteJob(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *AdminHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.adminUsecase.ListAllJobs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "jobs_list_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "jobs retrieved successfully", jobs)
}

func (h *AdminHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID string `json:"job_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	err := h.adminUsecase.DeleteJobListing(r.Context(), req.JobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "job_delete_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "job deleted successfully by administrator", map[string]string{
		"message": "job deleted successfully by administrator",
	})
}
