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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id != "" {
		userID, role, ok := GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		contract, err := h.contractUsecase.GetContract(r.Context(), userID, role, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(contract)
		return
	}

	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	contracts, err := h.contractUsecase.ListUserContracts(r.Context(), userID, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contracts)
}

func (h *ContractHandler) CompleteContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ContractID string `json:"contract_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.contractUsecase.CompleteContract(r.Context(), userID, req.ContractID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "contract marked completed successfully",
	})
}

func (h *ContractHandler) HandleDirectHires(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodPost:
		if role != "client" {
			http.Error(w, "unauthorized: only clients can create direct hire requests", http.StatusForbidden)
			return
		}
		var req struct {
			MusicianID  string  `json:"musician_id"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			Price       float64 `json:"price"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		directHire, err := h.contractUsecase.CreateDirectHireRequest(r.Context(), userID, req.MusicianID, req.Title, req.Description, req.Price)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(directHire)
	case http.MethodGet:
		requests, err := h.contractUsecase.ListDirectHireRequests(r.Context(), userID, role, r.URL.Query().Get("status"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(requests)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ContractHandler) RespondToDirectHire(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, role, ok := GetUserFromContext(r.Context())
	if !ok || role != "musician" {
		http.Error(w, "unauthorized: only musicians can respond to direct hire requests", http.StatusForbidden)
		return
	}
	var req struct {
		RequestID string `json:"request_id"`
		Decision  string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	directHire, err := h.contractUsecase.RespondToDirectHireRequest(r.Context(), userID, req.RequestID, req.Decision)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(directHire)
}
