package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/middleware/logging"
)

func NewRouter(h *handlers.Handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logging.WithLogging)

	r.Route("/api", func(r chi.Router) {
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", h.RegisterUser)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/devices", h.GetUserDevices)
				r.Get("/events", h.GetHistoryEvents)
			})
		})

		r.Route("/device", func(r chi.Router) {
			r.Post("/register", registerDeviceHandler)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getDeviceHandler)
				r.Delete("/", deleteDeviceHandler)
				r.Get("/events", getDeviceEventsHandler)
			})
		})

		r.Get("/health", healthHandler)
	})

	return r
}
