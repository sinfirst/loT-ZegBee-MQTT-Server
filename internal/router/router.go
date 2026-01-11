package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers/server"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/middleware/logging"
)

func NewRouter(h *server.HTTPServerHandlers) *chi.Mux {
	r := chi.NewRouter()
	r.Use(logging.WithLogging)

	r.Get("/api/user/{id}/devices", h.UserDevicesHandler)

	r.Route("/api", func(r chi.Router) {
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", h.RegisterUserHandler)
			r.Route("/{id}", func(r chi.Router) {
				//r.Get("/devices", h.UserDevicesHandler)
				r.Get("/events", h.HistoryEventsHandler)
			})
		})

		r.Route("/device", func(r chi.Router) {
			r.Post("/register", h.RegisterDeviceHandler)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.DeviceInfoHandler)
				r.Delete("/", h.DeleteDeviceHandler)
				r.Get("/events", h.DeviceHistoryHandler)
			})
		})

		r.Get("/health", h.HealthCheckHandler)
	})

	return r
}
