package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/middleware/logging"
)

func NewRouter(s *handlers.Handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logging.WithLogging)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// User endpoints
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", s.RegisterUserHandler)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/devices", s.getUserDevicesHandler)
				r.Get("/events", s.GetUserEventsHandler)
			})
		})

		// Device endpoints
		r.Route("/device", func(r chi.Router) {
			r.Post("/register", registerDeviceHandler)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getDeviceHandler)
				r.Delete("/", deleteDeviceHandler)
				r.Get("/events", getDeviceEventsHandler)
			})
		})

		// Hub endpoints
		r.Post("/hub/event", hubEventHandler)

		// System endpoints
		r.Get("/health", healthHandler)
	})

	return r
}
