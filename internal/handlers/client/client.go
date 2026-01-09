package client

import (
	"net/http"
	"time"

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
	return &ClientHandlers{
		logger:  logger,
		storage: storage,
		config:  config,
		client: http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}
