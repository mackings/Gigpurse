package http

import (
	"encoding/json"
	"net/http"

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

func (h *ContractHandler) HandleDirectHires(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		if role != "client" {
			respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only clients can create direct hire requests")
			return
		}
		var req struct {
			MusicianID  string  `json:"musician_id"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			Price       float64 `json:"price"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		directHire, err := h.contractUsecase.CreateDirectHireRequest(r.Context(), userID, req.MusicianID, req.Title, req.Description, req.Price)
		if err != nil {
			respondError(w, http.StatusBadRequest, "direct_hire_create_failed", err.Error())
			return
		}
		respondSuccess(w, http.StatusCreated, "direct hire request created successfully", directHire)
	case http.MethodGet:
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
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		respondError(w, http.StatusForbidden, "forbidden", "unauthorized: only musicians can respond to direct hire requests")
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
