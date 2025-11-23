package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/http"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/middleware/logging"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/storage"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	conf, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Panic("can't init config", err)
	}

	err = storage.InitMigrations(conf.DataBase.DataBaseDSN)
	if err != nil {
		log.Panic("can't init migrations", err)
	}

	logger := logging.NewLogger(conf.Log.Level)
	db := storage.NewPGDB(conf, logger)
	http := http.NewHTTPServer(logger)
	router := router.NewRouter(a)

	server := &http.Server{Addr: conf.ServerAddress, Handler: router}
	if !conf.HTTPSEnable {
		go func() {
			logger.Infow("Starting http server", "addr", conf.ServerAddress)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatalw("create server error: ", err)
			}
		}()
	} else {
		go func() {
			logger.Infow("Starting https server")
			err := http.ListenAndServeTLS(":8443", certFile, keyFile, nil)
			if err != nil {
				logger.Fatal("error while start server: ", err)
			}
		}()
	}

	<-ctx.Done()
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Errorw("Server shutdown error", err)
	}
}
