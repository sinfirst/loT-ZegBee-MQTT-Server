package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (h *HTTPServerHandlers) RegisterDeviceHandler(w http.ResponseWriter, r *http.Request) {
	type deviceRegisterRequest struct {
		UserID string `json:"user_id"`
		HubID  string `json:"hub_id"`
	}

	type statusSuccess struct {
		Status    string   `json:"status"`
		DevicesID []string `json:"devices_id"`
		Message   string   `json:"message"`
	}

	var req deviceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.responseWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		h.responseWithError(w, "UserID is required", http.StatusBadRequest)
		return
	}
	if req.HubID == "" {
		h.responseWithError(w, "HubID is required", http.StatusBadRequest)
		return
	}

	exist, err := h.storage.UserExistsByUserID(r.Context(), req.UserID)
	if err != nil {
		h.logger.Errorw("Failed to check user existence", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exist {
		h.responseWithError(w, "User not found", http.StatusNotFound)
		return
	}

	hubConnected, err := h.storage.ConnectExistByHubID(r.Context(), req.HubID)
	if err != nil {
		h.logger.Errorw("Failed to check hub connection", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if hubConnected {
		h.responseWithError(w, "Hub already registered to another user", http.StatusConflict)
		return
	}

	if err := h.mqttFunc.SubscribeToHub(req.HubID); err != nil {
		h.logger.Errorw("Failed to subscribe to hub MQTT topics",
			"hub_id", req.HubID,
			"error", err,
		)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	devicesID, err := h.storage.CreateConnect(r.Context(), req.UserID, req.HubID)
	if err != nil {
		h.logger.Errorw("Failed to create connection", "error", err)

		if unsubscribeErr := h.mqttFunc.UnsubscribeFromHub(req.HubID); unsubscribeErr != nil {
			h.logger.Errorw("Failed to unsubscribe after DB error",
				"hub_id", req.HubID,
				"error", unsubscribeErr,
			)
		}

		h.responseWithError(w, "Failed to register hub", http.StatusInternalServerError)
		return
	}

	message := "Hub registered successfully"
	if len(devicesID) == 0 {
		message = "Hub registered successfully. Devices will be added when detected. Check api/user/devices"
		h.logger.Infow("Hub registered, waiting for devices",
			"hub_id", req.HubID,
			"user_id", req.UserID,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:    "ok",
		DevicesID: devicesID,
		Message:   message,
	})
	if err != nil {
		h.logger.Errorw("Failed to encode response", "error", err)
	}
}

func (h *HTTPServerHandlers) DeviceInfoHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string        `json:"status"`
		Device models.Device `json:"device"`
	}

	path := r.URL.Path

	prefix := "/api/device/"
	if !strings.HasPrefix(path, prefix) {
		h.responseWithError(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deviceID := strings.TrimPrefix(path, prefix)

	deviceID = strings.TrimSuffix(deviceID, "/")

	fmt.Println(r.URL.Query())
	fmt.Println(deviceID)
	device, err := h.storage.GetDeviceInfo(r.Context(), deviceID)
	if err != nil {
		h.logger.Errorw("Failed to get device info", "error", err)
		h.responseWithError(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status: "ok",
		Device: device,
	})
	if err != nil {
		h.logger.Errorw("Failed to encode response", "error", err)
	}
}

func (h *HTTPServerHandlers) DeviceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string         `json:"status"`
		Events []models.Event `json:"events"`
	}

	deviceID := chi.URLParam(r, "id")
	hours := r.URL.Query().Get("hours")
	if hours == "" {
		hours = "24"
	}

	exist, err := h.storage.DeviceExistByDeviceID(r.Context(), deviceID)
	if err != nil {
		h.logger.Errorw("Failed to check device existence", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exist {
		h.responseWithError(w, "Device not found", http.StatusNotFound)
		return
	}

	events, err := h.storage.GetEventsByDeviceID(r.Context(), deviceID, hours)
	if err != nil {
		h.logger.Errorw("Failed to get device events", "error", err)
		h.responseWithError(w, "Failed to get events", http.StatusInternalServerError)
		return
	}

	if len(events) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(statusSuccess{
			Status: "ok",
			Events: []models.Event{},
		})
		if err != nil {
			h.logger.Errorw("Failed to encode response", "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status: "ok",
		Events: events,
	})
	if err != nil {
		h.logger.Errorw("Failed to encode response", "error", err)
	}
}

func (h *HTTPServerHandlers) DeleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	deviceID := chi.URLParam(r, "id")
	exist, err := h.storage.DeviceExistByDeviceID(r.Context(), deviceID)
	if err != nil {
		h.logger.Errorw("Failed to check device existence", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exist {
		h.responseWithError(w, "Device not found", http.StatusNotFound)
		return
	}

	err = h.storage.DeleteDevice(r.Context(), deviceID)
	if err != nil {
		h.logger.Errorw("Failed to delete device", "error", err)
		h.responseWithError(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:  "ok",
		Message: "Device deleted successfully",
	})
	if err != nil {
		h.logger.Errorw("Failed to encode response", "error", err)
	}
}
