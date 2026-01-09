package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers/client"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers/server"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/middleware/logging"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/mqtt"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/router"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
)

func main() {
	conf, err := config.LoadConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	logger := logging.NewLogger(conf.Log.Level)
	logger.Info("Configuration loaded successfully")

	logger.Info("Connecting to database")
	db := storage.NewPGDB(conf, logger)
	if db == nil {
		logger.Fatal("Failed to connect to database")
	}

	logger.Info("Initializing database migrations")
	if err := storage.InitMigrations(conf); err != nil {
		logger.Fatalw("Failed to initialize migrations", "error", err)
	}

	logger.Info("Database connected successfully")

	HTTPClient := client.NewClientHandlersStruct(logger, db, conf)

	logger.Infow("Starting MQTT client", "broker", conf.MQTT.Broker)
	mqttClient, err := mqtt.NewMQTTClient(conf, logger, HTTPClient)
	if err != nil {
		logger.Fatalw("Failed to initialize MQTT client", "error", err)
	}

	logger.Info("Restoring MQTT subscriptions for active hubs")
	if activeHubs, err := db.GetActiveHubs(context.Background()); err == nil && len(activeHubs) > 0 {
		mqttClient.RestoreSubscriptions(activeHubs)
		logger.Infow("Restored subscriptions", "hubs_count", len(activeHubs))
	} else if err != nil {
		logger.Warnw("Failed to get active hubs", "error", err)
	}

	logger.Info("MQTT client initialized successfully")

	HTTPServerHandlers := server.NewHTTPServerHandlers(logger, db, mqttClient)

	router := router.NewRouter(HTTPServerHandlers)

	HTTPServer := &http.Server{
		Addr:         conf.HTTP.Address,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Infow("Starting HTTP server", "addr", conf.HTTP.Address)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		if err := HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		logger.Fatalw("HTTP server error", "error", err)
	case <-stop:
		logger.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if mqttClient != nil {
			mqttClient.Close()
		}

		if err := HTTPServer.Shutdown(ctx); err != nil {
			logger.Errorw("Failed to shutdown HTTP server gracefully", "error", err)
		}

		logger.Info("Server stopped successfully")
	}
}
