package http

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success    bool        `json:"success"`
	Status     string      `json:"status"`
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Error      *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondSuccess(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success:    true,
		Status:     "success",
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	})
}

func respondError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success:    false,
		Status:     "error",
		StatusCode: statusCode,
		Message:    message,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}
