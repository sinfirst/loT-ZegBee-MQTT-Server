package server

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
		h.responseWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.TelegramID <= 0 {
		h.responseWithError(w, "Invalid telegram_id", http.StatusBadRequest)
		return
	}
	if req.Username == "" {
		h.responseWithError(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Проверка существования пользователя
	exist, userID, err := h.storage.UserExistsByTGID(r.Context(), req.TelegramID)
	if err != nil {
		h.logger.Errorw("Failed to check user existence", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if exist {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)

		err = json.NewEncoder(w).Encode(statusSuccess{
			Status:  "ok",
			UserID:  userID,
			Message: "User registered successfully",
		})
		if err != nil {
			h.logger.Errorw("Failed to encode response", "error", err)
		}
		return
	}

	// Создание пользователя
	userID, err = h.storage.CreateUser(r.Context(), req.TelegramID, req.Username)
	if err != nil {
		h.logger.Errorw("Failed to create user", "error", err)
		h.responseWithError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:  "ok",
		UserID:  userID,
		Message: "User registered successfully",
	})
	if err != nil {
		h.logger.Errorw("Failed to encode response", "error", err)
	}
}

func (h *HTTPServerHandlers) UserDevicesHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status  string          `json:"status"`
		Devices []models.Device `json:"devices"`
	}

	userID := chi.URLParam(r, "id")

	// Проверка существования пользователя
	exist, err := h.storage.UserExistsByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Errorw("Failed to check user existence", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exist {
		h.responseWithError(w, "User not found", http.StatusNotFound)
		return
	}

	// Получение устройств пользователя
	devices, err := h.storage.GetDevicesByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Errorw("Failed to get user devices", "error", err)
		h.responseWithError(w, "Failed to get devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(statusSuccess{
		Status:  "ok",
		Devices: devices,
	})
	if err != nil {
		h.logger.Errorw("Failed to encode response", "error", err)
	}
}

func (h *HTTPServerHandlers) HistoryEventsHandler(w http.ResponseWriter, r *http.Request) {
	type statusSuccess struct {
		Status string         `json:"status"`
		Events []models.Event `json:"events"`
	}

	userID := chi.URLParam(r, "id")
	hours := r.URL.Query().Get("hours")
	if hours == "" {
		hours = "24"
	}

	// Проверка существования пользователя
	exist, err := h.storage.UserExistsByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Errorw("Failed to check user existence", "error", err)
		h.responseWithError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !exist {
		h.responseWithError(w, "User not found", http.StatusNotFound)
		return
	}

	// Получение событий пользователя
	events, err := h.storage.GetEventsByUserID(r.Context(), userID, hours)
	if err != nil {
		h.logger.Errorw("Failed to get user events", "error", err)
		h.responseWithError(w, "Failed to get events", http.StatusInternalServerError)
		return
	}

	// Если событий нет
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
