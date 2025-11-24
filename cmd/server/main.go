package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/http"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/middleware/logging"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/mqtt"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/router"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	conf, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Panic("can't init config", err)
	}
	logger := logging.NewLogger(conf.Log.Level)
	logger.Info("Init config successfull")

	err = storage.InitMigrations(conf)
	if err != nil {
		logger.Fatal("can't init migrations", err)
	}
	logger.Info("Init config successfull")

	db := storage.NewPGDB(conf, logger)
	handlers := handlers.NewHandlersStruct(logger, db)
	mqtt := mqtt.NewMQTTClient(conf, logger, handlers)
	http := http.NewHTTPServer(logger)
	router := router.NewRouter(http)

	server := &http.Server{Addr: conf.HTTP.Address, Handler: router}
	go func() {
		logger.Infow("Starting http server", "addr", conf.HTTP.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalw("create server error: ", err)
		}
	}()

	<-ctx.Done()
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Errorw("Server shutdown error", err)
	}
}
