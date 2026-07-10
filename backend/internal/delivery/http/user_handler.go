package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gigpurse/internal/domain"
)

type UserHandler struct {
	userUsecase  domain.UserUsecase
	contractRepo domain.ContractRepository
}

func NewUserHandler(uu domain.UserUsecase, contractRepo domain.ContractRepository) *UserHandler {
	return &UserHandler{
		userUsecase:  uu,
		contractRepo: contractRepo,
	}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/signup", h.SignUp)
	mux.HandleFunc("/auth/login", h.Login)
	mux.HandleFunc("/auth/email-verification/resend", h.ResendEmailVerification)
	mux.HandleFunc("/auth/email-verification/confirm", h.VerifyEmail)
	mux.HandleFunc("/auth/password-reset/request", h.RequestPasswordReset)
	mux.HandleFunc("/auth/password-reset/confirm", h.ResetPassword)
	mux.HandleFunc("GET /users/profile", h.HandleProfile)
	mux.HandleFunc("PUT /users/profile", h.HandleProfile)
	mux.HandleFunc("GET /users/{id}", JWTMiddleware(h.GetUserByID))
	mux.HandleFunc("/musicians", h.BrowseMusicians)
	mux.HandleFunc("GET /musicians/{id}", h.GetMusicianByID)
}

func (h *UserHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		Role          string `json:"role"`
		Name          string `json:"name"`
		AcceptedTerms bool   `json:"accepted_terms"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	user, err := h.userUsecase.SignUp(r.Context(), req.Email, req.Password, req.Role, req.Name, req.AcceptedTerms)
	if err != nil {
		respondError(w, http.StatusBadRequest, "signup_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusCreated, "signup successful. verify your email before login", user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	token, user, err := h.userUsecase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "login_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "login successful", map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func (h *UserHandler) ResendEmailVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.userUsecase.ResendEmailVerification(r.Context(), req.Email); err != nil {
		respondError(w, http.StatusBadRequest, "email_verification_resend_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "if the email exists and is unverified, a verification message has been sent", nil)
}

func (h *UserHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.userUsecase.VerifyEmail(r.Context(), req.Email, req.Code); err != nil {
		respondError(w, http.StatusBadRequest, "email_verification_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "email verified successfully", nil)
}

func (h *UserHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.userUsecase.RequestPasswordReset(r.Context(), req.Email); err != nil {
		respondError(w, http.StatusBadRequest, "password_reset_request_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "if the email exists, a password reset message has been sent", nil)
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	if err := h.userUsecase.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		respondError(w, http.StatusBadRequest, "password_reset_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "password reset successfully", nil)
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
			respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})(w, r)
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := h.userUsecase.GetProfile(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "profile_not_found", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "profile retrieved successfully", user)
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
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	user, err := h.userUsecase.UpdateProfile(r.Context(), userID, req.Name, req.Bio, req.Location, req.MusicianProfile, req.ClientProfile)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "profile_update_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "profile updated successfully", user)
}

func (h *UserHandler) BrowseMusicians(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
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
		respondError(w, http.StatusInternalServerError, "musician_search_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "musicians retrieved successfully", musicians)
}

// GetUserByID returns a minimal, non-sensitive projection (id/name/role) of
// any user, regardless of role — used to resolve display names for chat
// partners (a client and a musician can both appear on either side of a
// conversation, and only musicians have a public profile endpoint).
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user, err := h.userUsecase.GetProfile(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	respondSuccess(w, http.StatusOK, "user retrieved successfully", map[string]any{
		"id":             user.ID,
		"name":           user.Name,
		"role":           user.Role,
		"location":       user.Location,
		"created_at":     user.CreatedAt,
		"client_profile": user.ClientProfile,
	})
}

func (h *UserHandler) GetMusicianByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user, err := h.userUsecase.GetProfile(r.Context(), id)
	if err != nil || user.Role != "musician" {
		respondError(w, http.StatusNotFound, "musician_not_found", "musician not found")
		return
	}

	// Public trust stats (completed gig count + total earned), shown on the
	// profile the same way marketplaces like Upwork surface "$X earned" —
	// best-effort only, a contract lookup failure shouldn't 404 the profile.
	if contracts, err := h.contractRepo.ListForUser(r.Context(), id, "musician"); err == nil {
		for _, c := range contracts {
			if c.Status == "completed" {
				user.CompletedContracts++
				user.TotalEarned += c.Price
			}
		}
	}

	respondSuccess(w, http.StatusOK, "musician retrieved successfully", user)
}
