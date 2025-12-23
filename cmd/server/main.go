package main

import (
	"net/http"

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
	logger.Info("Init config successfull")

	db := storage.NewPGDB(conf, logger)
	err = storage.InitMigrations(conf)
	logger.Infow("Starting init migrations")
	if err != nil {
		logger.Fatal("can't init migrations", err)
	}

	HTTPClient := client.NewClientHandlersStruct(logger, db, conf)
	mqttClient := mqtt.NewMQTTClient(conf, logger, HTTPClient)
	logger.Infow("Starting mqtt client", "broker", conf.MQTT.Broker)
	if err := mqttClient.Connect(); err != nil {
		logger.Fatal("can't init mqtt connect", err)
	}

	HTTPServerHandlers := server.NewHTTPServerHandlers(logger, db)
	router := router.NewRouter(HTTPServerHandlers)
	HTTPServer := &http.Server{Addr: conf.HTTP.Address, Handler: router}
	logger.Infow("Starting http server", "addr", conf.HTTP.Address)
	if err := HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalw("can't init http server: ", err)
	}

}
