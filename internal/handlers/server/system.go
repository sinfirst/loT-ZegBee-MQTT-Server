package server

import (
	"encoding/json"
	"net/http"
)

func (h *HTTPServerHandlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string `json:"status"`
	}

	// some checks

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(statusSuccess{
		Status: "ok",
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
	}
}
