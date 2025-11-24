package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type RegisterUserRequest struct {
	TelegramID int64  `json:"telegram_id"`
	Username   string `json:"username"`
}

func (h *Handlers) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if h.storage.UserExists(req.TelegramID) {
		respondWithError(w, http.StatusConflict, "User already exists")
		return
	}

	user := &models.User{
		UserID:     "user_" + strconv.FormatInt(req.TelegramID, 10),
		TelegramID: req.TelegramID,
		Username:   req.Username,
	}

	if err := h.storage.CreateUser(user); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	respondWithJSON(w, http.StatusOK, models.Response{
		Status:  "ok",
		Message: "User registered",
		Data:    map[string]string{"user_id": user.UserID},
	})
}
