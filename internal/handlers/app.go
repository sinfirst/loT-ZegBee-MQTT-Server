package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
	"go.uber.org/zap"
)

type Handlers struct {
	logger  *zap.SugaredLogger
	storage *storage.PGDB
}

func NewHandlersStruct(logger *zap.SugaredLogger, storage *storage.PGDB) *Handlers {
	return &Handlers{logger: logger, storage: storage}
}

func (h *Handlers) NewDiviceMessageHandler(msg []byte) []string {

}

func (h *Handlers) responseWithError(w http.ResponseWriter, message string, status int) {
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
