package handlers

import (
	"encoding/json"
	"net/http"

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
		h.responseWithError(w, "Bad request", http.StatusBadRequest)
		return
	}

	exist, err := h.storage.UserExistsByUserID(r.Context(), req.UserID)
	if err != nil {
		h.responseWithError(w, "Failed to check user exist in DB", http.StatusInternalServerError)
		return
	}
	if exist == false {
		h.responseWithError(w, "User not found", http.StatusNotFound)
		return
	}

	exist, err = h.storage.ConnectExistByHubID(r.Context(), req.HubID)
	if err != nil {
		h.responseWithError(w, "Failed to check user exist in DB", http.StatusInternalServerError)
		return
	}
	if exist {
		h.responseWithError(w, "Devices already registered", http.StatusConflict)
		return
	}

	devicesID, err := h.storage.CreateConnect(r.Context(), req.UserID, req.HubID)
	if err != nil {
		h.responseWithError(w, "Failed to create connect", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:    "ok",
		DevicesID: devicesID,
		Message:   "Device registered",
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
	}
}

func (h *HTTPServerHandlers) DeviceInfoHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string        `json:"status"`
		Device models.Device `json:"device"`
	}
	deviceID := chi.URLParam(r, "id")

	exist, err := h.storage.DeviceExistByDeviceID(r.Context(), deviceID)
	if err != nil {
		h.responseWithError(w, "Failed to check device exist in DB", http.StatusInternalServerError)
		return
	}
	if exist == false {
		h.responseWithError(w, "Device not found", http.StatusNotFound)
		return
	}

	device, err := h.storage.GetDeviceInfo(r.Context(), deviceID)
	if err != nil {
		h.responseWithError(w, "Failed to get info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status: "ok",
		Device: device,
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
	}
}

func (h *HTTPServerHandlers) DeviceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string         `json:"status"`
		Events []models.Event `json:"devices"`
	}

	deviceID := chi.URLParam(r, "id")

	hours := r.URL.Query().Get("hours")
	if hours == "" {
		hours = "24"
	}

	exist, err := h.storage.DeviceExistByDeviceID(r.Context(), deviceID)
	if err != nil {
		h.responseWithError(w, "Failed to check device exist in DB", http.StatusInternalServerError)
		return
	}
	if exist == false {
		h.responseWithError(w, "Device not found", http.StatusNotFound)
		return
	}

	events, err := h.storage.GetEventsByDeviceID(r.Context(), deviceID, hours)
	if err != nil {
		h.responseWithError(w, "Failed found events", http.StatusInternalServerError)
		return
	}

	if events == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)

		err = json.NewEncoder(w).Encode(statusSuccess{
			Status: "ok",
		})
		if err != nil {
			h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status: "ok",
		Events: events,
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
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
		h.responseWithError(w, "Failed to check device exist in DB", http.StatusInternalServerError)
		return
	}
	if exist == false {
		h.responseWithError(w, "Device not found", http.StatusNotFound)
		return
	}

	err = h.storage.DeleteDevice(r.Context(), deviceID)
	if err != nil {
		h.responseWithError(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:  "ok",
		Message: "Device removed",
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
	}
}
