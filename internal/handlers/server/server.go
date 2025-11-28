package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
	"go.uber.org/zap"
)

type HTTPServerHandlers struct {
	logger  *zap.SugaredLogger
	storage *storage.PGDB
}

func NewHTTPServerHandlers(logger *zap.SugaredLogger, storage *storage.PGDB) *HTTPServerHandlers {
	return &HTTPServerHandlers{logger: logger, storage: storage}
}

func (h *HTTPServerHandlers) responseWithError(w http.ResponseWriter, message string, status int) {
	type statusError struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	resp := statusError{
		Status:  "error",
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Warnf("err with response error: %v", err)
		w.WriteHeader(status)
		return
	}
}
