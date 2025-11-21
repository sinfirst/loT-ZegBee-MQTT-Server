package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	conf, err := config.NewConfig()
	if err != nil {
		logger.Fatalw("can't init config", err)
	}

	logger := logging.NewLogger(conf.LogLvl)
	db := postgresbd.NewPGDB(conf, logger)

	a := app.NewHTTPServer(logger)
	router := router.NewRouter(a)

	if conf.DatabaseDsn != "" {
		err := postgresbd.InitMigrations(conf, logger)
		if err != nil {
			logger.Fatalw("can't init migrations", err)
		}
	}
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

	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(logging.LoggingUnaryInterceptor(logger)))
	pb.RegisterURLCutterServer(s, grpcserver.NewURLCutterServer(logger, handlers))
	fmt.Println("Сервер gRPC начал работу")
	if err := s.Serve(listen); err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Errorw("Server shutdown error", err)
	}
	workers.StopWorker()
	close(deleteCh)
}
