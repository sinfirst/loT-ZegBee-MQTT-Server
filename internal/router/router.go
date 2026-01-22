package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers/server"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/middleware/logging"
)

func NewRouter(h *server.HTTPServerHandlers) *chi.Mux {
	r := chi.NewRouter()
	r.Use(logging.WithLogging)

	// User endpoints
	r.Post("/api/user/register", h.RegisterUserHandler)
	r.Get("/api/user/{id}/devices", h.UserDevicesHandler)
	r.Get("/api/user/{id}/events", h.HistoryEventsHandler)

	// Device endpoints
	r.Get("/api/device/{id}", h.DeviceInfoHandler)
	r.Get("/api/device/{id}/events", h.DeviceHistoryHandler)

	//Hub endpoints
	r.Post("/api/hub/register", h.RegisterHubHandler)
	r.Delete("/api/hub/{id}", h.DeleteHubHandler)

	// Health check
	r.Get("/api/health", h.HealthCheckHandler)

	return r
}
