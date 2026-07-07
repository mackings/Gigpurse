package http

import (
	"encoding/json"
	"net/http"
	"time"

	"gigpurse/internal/domain"
)

type ContractHandler struct {
	contractUsecase domain.ContractUsecase
}

func NewContractHandler(cu domain.ContractUsecase) *ContractHandler {
	return &ContractHandler{
		contractUsecase: cu,
	}
}

func (h *ContractHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/contracts", JWTMiddleware(h.HandleContracts))
	mux.HandleFunc("/contracts/complete", JWTMiddleware(h.CompleteContract))
	mux.HandleFunc("/direct-hires", JWTMiddleware(h.HandleDirectHires))
	mux.HandleFunc("/direct-hires/respond", JWTMiddleware(h.RespondToDirectHire))
	mux.HandleFunc("/direct-hires/counter", JWTMiddleware(h.CounterDirectHire))
}

// parseEventDate parses an optional RFC3339 date string; empty is not an error.
func parseEventDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *ContractHandler) HandleContracts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	id := r.URL.Query().Get("id")
	if id != "" {
		userID, role, ok := GetUserFromContext(r.Context())
		if !ok {
			respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}
		contract, err := h.contractUsecase.GetContract(r.Context(), userID, role, id)
		if err != nil {
			respondError(w, http.StatusForbidden, "contract_access_denied", err.Error())
			return
		}
		respondSuccess(w, http.StatusOK, "contract retrieved successfully", contract)
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	contracts, err := h.contractUsecase.ListUserContracts(r.Context(), userID, role)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "contracts_list_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "contracts retrieved successfully", contracts)
}

func (h *ContractHandler) CompleteContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req struct {
		ContractID string `json:"contract_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	err := h.contractUsecase.CompleteContract(r.Context(), userID, req.ContractID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "contract_complete_failed", err.Error())
		return
	}

	respondSuccess(w, http.StatusOK, "contract marked completed successfully", map[string]string{
		"message": "contract marked completed successfully",
	})
}

type directHireTermsRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	EventDate   string  `json:"event_date"`
	Price       float64 `json:"price"`
}

func (h *ContractHandler) HandleDirectHires(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		if role != "client" && role != "musician" {
			respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only clients or musicians can create direct hire requests")
			return
		}
		var req struct {
			directHireTermsRequest
			MusicianID   string `json:"musician_id"`
			TargetUserID string `json:"target_user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		counterpartID := req.TargetUserID
		if counterpartID == "" {
			counterpartID = req.MusicianID
		}
		eventDate, err := parseEventDate(req.EventDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_event_date", "event_date must be RFC3339")
			return
		}
		directHire, err := h.contractUsecase.CreateDirectHireRequest(r.Context(), userID, counterpartID, domain.DirectHireTerms{
			Title: req.Title, Description: req.Description, Location: req.Location, EventDate: eventDate, Price: req.Price,
		})
		if err != nil {
			respondError(w, http.StatusBadRequest, "direct_hire_create_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusCreated, "direct hire request created successfully", directHire)
	case http.MethodGet:
		if id := r.URL.Query().Get("id"); id != "" {
			directHire, err := h.contractUsecase.GetDirectHireRequest(r.Context(), userID, id)
			if err != nil {
				respondError(w, http.StatusForbidden, "direct_hire_access_denied", err.Error())
				return
			}
			respondSuccess(w, http.StatusOK, "direct hire request retrieved successfully", directHire)
			return
		}
		requests, err := h.contractUsecase.ListDirectHireRequests(r.Context(), userID, role, r.URL.Query().Get("status"))
		if err != nil {
			respondError(w, http.StatusInternalServerError, "direct_hires_list_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusOK, "direct hire requests retrieved successfully", requests)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *ContractHandler) RespondToDirectHire(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req struct {
		RequestID string `json:"request_id"`
		Decision  string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	directHire, err := h.contractUsecase.RespondToDirectHireRequest(r.Context(), userID, req.RequestID, req.Decision)
	if err != nil {
		respondError(w, http.StatusBadRequest, "direct_hire_response_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "direct hire response saved successfully", directHire)
}

func (h *ContractHandler) CounterDirectHire(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req struct {
		directHireTermsRequest
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	eventDate, err := parseEventDate(req.EventDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_event_date", "event_date must be RFC3339")
		return
	}
	directHire, err := h.contractUsecase.CounterDirectHireRequest(r.Context(), userID, req.RequestID, domain.DirectHireTerms{
		Title: req.Title, Description: req.Description, Location: req.Location, EventDate: eventDate, Price: req.Price,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "direct_hire_counter_failed", err.Error())
		return
	}
	respondSuccess(w, http.StatusOK, "counter-offer saved successfully", directHire)
}
