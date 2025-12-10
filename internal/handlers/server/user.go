package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (h *HTTPServerHandlers) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	type registerUserRequest struct {
		TelegramID int    `json:"telegram_id"`
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

	exist, err := h.storage.UserExistsByTGID(r.Context(), req.TelegramID)
	if err != nil {
		h.responseWithError(w, "Failed to check user exist in DB", http.StatusInternalServerError)
		return
	}
	if exist {
		h.responseWithError(w, "User alredy exist", http.StatusConflict)
		return
	}

	userID, err := h.storage.CreateUser(r.Context(), req.TelegramID, req.Username)
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

func (h *HTTPServerHandlers) UserDevicesHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status  string          `json:"status"`
		Devices []models.Device `json:"devices"`
	}

	userID := chi.URLParam(r, "id")

	exist, err := h.storage.UserExistsByUserID(r.Context(), userID)
	if err != nil {
		h.responseWithError(w, "Failed to check user exist in DB", http.StatusInternalServerError)
		return
	}
	if exist == false {
		h.responseWithError(w, "User not found", http.StatusNotFound)
		return
	}

	devices, err := h.storage.GetDevicesByUserID(r.Context(), userID)
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

func (h *HTTPServerHandlers) HistoryEventsHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string         `json:"status"`
		Events []models.Event `json:"devices"`
	}

	userID := chi.URLParam(r, "id")

	hours := r.URL.Query().Get("hours")
	if hours == "" {
		hours = "24"
	}

	exist, err := h.storage.UserExistsByUserID(r.Context(), userID)
	if err != nil {
		h.responseWithError(w, "Failed to check user exist in DB", http.StatusInternalServerError)
		return
	}
	if exist == false {
		h.responseWithError(w, "User not found", http.StatusNotFound)
		return
	}

	events, err := h.storage.GetEventsByUserID(r.Context(), userID, hours)
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
