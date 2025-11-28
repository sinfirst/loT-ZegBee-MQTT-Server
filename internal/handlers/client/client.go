package handlers

import (
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
	"go.uber.org/zap"
)

type ClientHandlers struct {
	logger  *zap.SugaredLogger
	storage *storage.PGDB
}

func NewClientHandlersStruct(logger *zap.SugaredLogger, storage *storage.PGDB) *ClientHandlers {
	return &ClientHandlers{logger: logger, storage: storage}
}
