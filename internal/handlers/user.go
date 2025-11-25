package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (h *Handlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
	type registerUserRequest struct {
		TelegramID int64  `json:"telegram_id"`
		Username   string `json:"username"`
	}

	type statusSuccess struct {
		Status  string `json:"status"`
		UserID  string `json:"user_id"`
		Message string `json:"message"`
	}

	var req registerUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.responseWithError(w, "Bad request", http.StatusBadRequest)
		return
	}

	if h.storage.UserExistsByTGID(req.TelegramID) {
		h.responseWithError(w, "User alredy exist", http.StatusConflict)
		return
	}

	userID, err := h.storage.CreateUser(user)
	if err != nil {
		h.responseWithError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:  "ok",
		UserID:  userID,
		Message: "Device registered",
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) GetUserDevices(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status  string          `json:"status"`
		Devices []models.Device `json:"devices"`
	}

	userID := chi.URLParam(r, "id")

	if h.storage.UserExistsByUserID(userID) == false {
		h.responseWithError(w, "User doesn't exist", http.StatusNotFound)
		return
	}

	devices, err := h.storage.GetDevicesByUserID(userID)
	if err != nil {
		h.responseWithError(w, "Failed found devices", http.StatusInternalServerError)
		return
	}

	if devices == nil {
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
		Status:  "ok",
		Devices: devices,
	})
	if err != nil {
		h.responseWithError(w, "Failed to response answer", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) GetHistoryEvents(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string         `json:"status"`
		Events []models.Event `json:"devices"`
	}

	userId := chi.URLParam(r, "id")

	hours := r.URL.Query().Get("hours")
	if hours == "" {
		hours = "24"
	}

	if h.storage.UserExistsByUserID(userId) == false {
		h.responseWithError(w, "User doesn't exist", http.StatusNotFound)
		return
	}

	events, err := h.storage.GetEventsByUserID(userId)
	if err != nil {
		h.responseWithError(w, "Failed found devices", http.StatusInternalServerError)
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
