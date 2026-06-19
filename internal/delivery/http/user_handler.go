package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gigpurse/internal/domain"
)

type UserHandler struct {
	userUsecase domain.UserUsecase
}

func NewUserHandler(uu domain.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: uu,
	}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/signup", h.SignUp)
	mux.HandleFunc("/auth/login", h.Login)
	mux.HandleFunc("/auth/password-reset/request", h.RequestPasswordReset)
	mux.HandleFunc("/auth/password-reset/confirm", h.ResetPassword)
	mux.HandleFunc("/users/profile", h.HandleProfile)
	mux.HandleFunc("/musicians", h.BrowseMusicians)
}

func (h *UserHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userUsecase.SignUp(r.Context(), req.Email, req.Password, req.Role, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, user, err := h.userUsecase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func (h *UserHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.userUsecase.RequestPasswordReset(r.Context(), req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "if the email exists, a password reset message has been sent"})
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.userUsecase.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "password reset successfully"})
}

func (h *UserHandler) HandleProfile(w http.ResponseWriter, r *http.Request) {
	JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID, _, _ := GetUserFromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			h.GetProfile(w, r, userID)
		case http.MethodPut:
			h.UpdateProfile(w, r, userID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})(w, r)
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := h.userUsecase.GetProfile(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request, userID string) {
	var req struct {
		Name            string                  `json:"name"`
		Bio             string                  `json:"bio"`
		Location        string                  `json:"location"`
		MusicianProfile *domain.MusicianProfile `json:"musician_profile,omitempty"`
		ClientProfile   *domain.ClientProfile   `json:"client_profile,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userUsecase.UpdateProfile(r.Context(), userID, req.Name, req.Bio, req.Location, req.MusicianProfile, req.ClientProfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) BrowseMusicians(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	minExp, _ := strconv.Atoi(q.Get("min_exp"))

	filter := domain.MusicianFilter{
		Genre:      q.Get("genre"),
		Instrument: q.Get("instrument"),
		Location:   q.Get("location"),
		MinExp:     minExp,
		SortBy:     q.Get("sort_by"),
		SortOrder:  q.Get("sort_order"),
	}

	musicians, err := h.userUsecase.BrowseMusicians(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(musicians)
}
