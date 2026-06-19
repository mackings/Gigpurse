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
		http.Error(w, "forbidden: admin role required", http.StatusForbidden)
		return false
	}
	return true
}

func (h *AdminHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkAdmin(w, r) {
		return
	}

	analytics, err := h.adminUsecase.GetAnalytics(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkAdmin(w, r) {
		return
	}

	users, err := h.adminUsecase.ListAllUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.adminUsecase.ListAllJobs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (h *AdminHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID string `json:"job_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.adminUsecase.DeleteJobListing(r.Context(), req.JobID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "job deleted successfully by administrator",
	})
}
