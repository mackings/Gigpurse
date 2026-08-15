package http

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	mux.HandleFunc("GET /admin/jobs/{id}", JWTMiddleware(h.GetJobDetail))
	mux.HandleFunc("/admin/moderators", JWTMiddleware(h.HandleModerators))
	mux.HandleFunc("/admin/moderators/invite", JWTMiddleware(h.InviteModerator))
	mux.HandleFunc("/admin/moderators/status", JWTMiddleware(h.SetModeratorStatus))
	mux.HandleFunc("/admin/talent-engagement", JWTMiddleware(h.GetTalentEngagement))
	mux.HandleFunc("/admin/client-engagement", JWTMiddleware(h.GetClientEngagement))
}

// windowDays parses the shared ?window= query param (days), defaulting to
// 30 and rejecting anything nonsensical rather than silently clamping.
func windowDays(r *http.Request) int {
	raw := r.URL.Query().Get("window")
	if raw == "" {
		return 30
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 30
	}
	return n
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

func (h *AdminHandler) GetJobDetail(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdmin(w, r) {
		return
	}
	detail, err := h.adminUsecase.GetJobDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusNotFound, "job_not_found", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "job detail retrieved successfully", detail)
}

func (h *AdminHandler) GetTalentEngagement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.checkAdmin(w, r) {
		return
	}
	summaries, err := h.adminUsecase.ListTalentEngagement(r.Context(), windowDays(r))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "talent_engagement_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "talent engagement retrieved successfully", summaries)
}

func (h *AdminHandler) GetClientEngagement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if !h.checkAdmin(w, r) {
		return
	}
	summaries, err := h.adminUsecase.ListClientEngagement(r.Context(), windowDays(r))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "client_engagement_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "client engagement retrieved successfully", summaries)
}

func (h *AdminHandler) HandleModerators(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	moderators, err := h.adminUsecase.ListModerators(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "moderators_list_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "moderators retrieved successfully", moderators)
}

func (h *AdminHandler) InviteModerator(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	moderator, err := h.adminUsecase.InviteModerator(r.Context(), req.Email, req.Name)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invite_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "moderator invited successfully", moderator)
}

func (h *AdminHandler) SetModeratorStatus(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		UserID string `json:"user_id"`
		Active bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.adminUsecase.SetModeratorStatus(r.Context(), req.UserID, req.Active); err != nil {
		respondError(w, http.StatusBadRequest, "status_update_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "moderator status updated successfully", map[string]bool{"active": req.Active})
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
