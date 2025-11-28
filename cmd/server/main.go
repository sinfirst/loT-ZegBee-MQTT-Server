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

	HTTPServerHandlers := handlers.NewHTTPServerHandlers(logger, db)
	router := router.NewRouter(HTTPServerHandlers)
	HTTPServer := &http.Server{Addr: conf.HTTP.Address, Handler: router}
	go func() {
		logger.Infow("Starting http server", "addr", conf.HTTP.Address)
		if err := HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalw("create server error: ", err)
		}
	}()
	<-ctx.Done()
	if err := HTTPServer.Shutdown(context.Background()); err != nil {
		logger.Errorw("Server shutdown error", err)
	}

	mqttClient := mqtt.NewMQTTClient(conf, logger, HTTPServerHandlers)
}
