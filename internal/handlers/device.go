package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
)

func (h *Handlers) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	type deviceRegisterRequest struct {
		UserID   string `json:"user_id"`
		DeviceID string `json:"device_id"`
	}

	type statusSuccess struct {
		Status   string `json:"status"`
		DeviceID string `json:"device_id"`
		Message  string `json:"message"`
	}

	var req deviceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.responseWithError(w, "Bad request", http.StatusBadRequest)
		return
	}

	if h.storage.UserExistsByUserID(req.UserID) == false {
		h.responseWithError(w, "User doesn't exist", http.StatusNotFound)
		return
	}

	if h.storage.DeviceExistByDeviceID(req.DeviceID) == false {
		h.responseWithError(w, "Device doesn't exist", http.StatusNotFound)
		return
	}

	if h.storage.ConnectExistByDeviceID(req.DeviceID) {
		h.responseWithError(w, "Device already registered", http.StatusConflict)
		return
	}

	err := h.storage.CreateConnect(req.UserID, req.DeviceID)
	if err != nil {
		h.responseWithError(w, "Failed to create connect", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:   "ok",
		DeviceID: req.DeviceID,
		Message:  "Device registered",
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) DeviceInfo(w http.ResponseWriter, r *http.Request) {
	deviceId := chi.URLParam(r, "id")

	if h.storage.DeviceExistByDeviceID(deviceID) == false {
		h.responseWithError(w, "Device doesn't exist", http.StatusNotFound)
		return
	}

	

}
