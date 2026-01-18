package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (h *HTTPServerHandlers) RegisterHubHandler(w http.ResponseWriter, r *http.Request) {
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

	connectedUserID, err := h.storage.GetHubConnectedUser(r.Context(), req.HubID)
	if err != nil {
		h.logger.Errorw("Failed to check hub connection", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if connectedUserID != "" {
		if connectedUserID == req.UserID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			err = json.NewEncoder(w).Encode(statusSuccess{
				Status:    "ok",
				DevicesID: nil,
				Message:   "User already registred in this hub. Check api/user/devices.",
			})
			if err != nil {
				h.logger.Errorw("Failed to encode response", "error", err)
			}
			return
		} else {
			h.responseWithError(w, "Hub already registered to another user", http.StatusConflict)
			return
		}
	}

	if err := h.mqttFunc.SubscribeToHub(req.HubID); err != nil {
		h.logger.Errorw("Failed to subscribe to hub MQTT topics",
			"hub_id", req.HubID,
			"error", err,
		)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	//Костыль
	time.Sleep(1 * time.Second)

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
		message = "Hub registered successfully. If devices_id null, do request again. If it not help, problem in another things."
		h.logger.Infow("Hub registered, waiting for devices",
			"hub_id", req.HubID,
			"user_id", req.UserID,
		)
	}

	for _, deviceID := range devicesID {
		go h.notificator.StartPooler(deviceID, req.UserID)
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

	deviceID, err := h.getIDFromURLPath(r, "/api/device/")
	if err != nil {
		h.responseWithError(w, "Invalid path", http.StatusBadRequest)
		return
	}

	device, err := h.storage.GetDeviceInfo(r.Context(), deviceID)
	if err != nil {
		if err.Error() == "not found" {
			h.logger.Errorw("Failed to get device info", "error", err)
			h.responseWithError(w, "Device not found", http.StatusNotFound)
			return
		}
		h.logger.Errorw("Failed to get device info", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
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

	deviceID, err := h.getIDFromURLPath(r, "/api/device/")
	if err != nil {
		h.responseWithError(w, "Invalid path", http.StatusBadRequest)
		return
	}

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

func (h *HTTPServerHandlers) DeleteHubHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status         string   `json:"status"`
		Message        string   `json:"message"`
		DeletedDevices []string `json:"deleted_devices,omitempty"`
	}

	hubID, err := h.getIDFromURLPath(r, "/api/device/")
	if err != nil {
		h.responseWithError(w, "Invalid path", http.StatusBadRequest)
		return
	}

	devices, err := h.storage.GetHubDevices(r.Context(), hubID)
	if err != nil {
		h.logger.Errorw("Failed to get hub devices for deletion", "hub_id", hubID, "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.mqttFunc.UnsubscribeFromHub(hubID); err != nil {
		h.logger.Warnw("Failed to unsubscribe from hub MQTT topics",
			"hub_id", hubID, "error", err)
	}

	deletedCount, err := h.storage.DeleteHubDevices(r.Context(), hubID)
	if err != nil {
		h.logger.Errorw("Failed to delete hub devices", "hub_id", hubID, "error", err)
		h.responseWithError(w, "Failed to delete hub", http.StatusInternalServerError)
		return
	}

	if err := h.storage.UnassignHubFromUser(r.Context(), hubID); err != nil {
		h.logger.Errorw("Failed to unassign hub from user", "hub_id", hubID, "error", err)
	}

	h.logger.Infow("Hub deleted successfully",
		"hub_id", hubID,
		"deleted_devices", deletedCount)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:         "ok",
		Message:        "Hub and all its devices deleted successfully",
		DeletedDevices: devices,
	})
	if err != nil {
		h.logger.Errorw("Failed to encode response", "error", err)
	}
}
