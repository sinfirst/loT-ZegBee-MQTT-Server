package client

import (
	"net/http"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
	"go.uber.org/zap"
)

type ClientHandlers struct {
	logger  *zap.SugaredLogger
	storage *storage.PGDB
	config  *config.Config
	client  http.Client
}

func NewClientHandlersStruct(logger *zap.SugaredLogger, storage *storage.PGDB, config *config.Config) *ClientHandlers {
	return &ClientHandlers{logger: logger, storage: storage, config: config}
}
