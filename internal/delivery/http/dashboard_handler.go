package http

import (
	"net/http"

	"gigpurse/internal/domain"
)

type DashboardHandler struct {
	dashboardUsecase domain.DashboardUsecase
}

func NewDashboardHandler(du domain.DashboardUsecase) *DashboardHandler {
	return &DashboardHandler{dashboardUsecase: du}
}

func (h *DashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/talent/dashboard", JWTMiddleware(h.GetTalentDashboard))
}

func (h *DashboardHandler) GetTalentDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can view talent dashboard")
		return
	}

	dashboard, err := h.dashboardUsecase.GetTalentDashboard(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "dashboard_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "talent dashboard retrieved successfully", dashboard)
}
