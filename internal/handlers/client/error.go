package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// ErrorHandler отправляет уведомление об ошибке
func (c *ClientHandlers) ErrorHandler(hubID, deviceID, errorType, message string) error {
	type errorNotification struct {
		UserID    string    `json:"user_id,omitempty"`
		HubID     string    `json:"hub_id,omitempty"`
		DeviceID  string    `json:"device_id,omitempty"`
		ErrorType string    `json:"error_type"`
		Message   string    `json:"message"`
		Timestamp time.Time `json:"timestamp"`
	}

	type errorResponse struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	var userID string
	if hubID != "" {
		if uid, err := c.storage.GetHubUserID(context.Background(), hubID); err == nil {
			userID = uid
		}
	} else if deviceID != "" {
		if uid, err := c.storage.GetUserIDByDeviceID(context.Background(), deviceID); err == nil {
			userID = uid
		}
	}

	if userID == "" {
		c.logger.Warnw("No user found for error notification",
			"error_type", errorType,
			"message", message,
			"hub_id", hubID,
			"device_id", deviceID,
		)
		return nil
	}

	notification := errorNotification{
		UserID:    userID,
		HubID:     hubID,
		DeviceID:  deviceID,
		ErrorType: errorType,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}

	jsonData, err := json.Marshal(notification)
	if err != nil {
		c.logger.Errorw("Failed to marshal error notification", "error", err)
		return err
	}

	req, err := http.NewRequest("POST", c.config.HTTP.Address+"/api/bot/error", bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.Errorw("Failed to create error notification request", "error", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Error-Notification", "true")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Errorw("Failed to send error notification", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warnw("Server returned non-OK status for error notification",
			"status", resp.StatusCode,
		)
	}

	var botResp errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&botResp); err != nil {
		c.logger.Errorw("Failed to decode error response", "error", err)
		return err
	}

	if botResp.Status != "ok" {
		c.logger.Warnw("Bot returned error status", "message", botResp.Message)
	}

	return nil
}
