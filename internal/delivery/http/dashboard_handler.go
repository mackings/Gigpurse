package http

import (
	"encoding/json"
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		http.Error(w, "unauthorized: only musicians can view talent dashboard", http.StatusForbidden)
		return
	}

	dashboard, err := h.dashboardUsecase.GetTalentDashboard(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}
