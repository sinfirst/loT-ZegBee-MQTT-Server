package handlers

import (
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
	"go.uber.org/zap"
)

type Handlers struct {
	logger  zap.SugaredLogger
	storage storage.PGDB
}

func NewHandlersStruct(logger zap.SugaredLogger, storage storage.PGDB) *Handlers {
	return &Handlers{logger: logger, storage: storage}
}

func (h *Handlers) NewDiviceMessageHandler(msg []byte) []string {

}
