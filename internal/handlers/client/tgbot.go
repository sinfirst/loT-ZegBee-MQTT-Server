package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (c *ClientHandlers) pushEvent(event models.Event, userID string) error {
	type botNotification struct {
		UserID    string    `json:"user_id"`
		DeviceID  string    `json:"device_id"`
		Event     string    `json:"event"`
		Timestamp time.Time `json:"timestamp"`
	}

	type botResponse struct {
		Status string `json:"status"`
	}

	var eventType string
	if event.EventType != "" {
		eventType = event.EventType
	} else {
		eventType = "movement_detected"
	}

	notification := botNotification{
		UserID:    userID,
		DeviceID:  event.DeviceID,
		Event:     eventType,
		Timestamp: time.Now().UTC(),
	}

	jsonData, err := json.Marshal(notification)
	if err != nil {
		c.logger.Errorw("Failed to marshal notification", "error", err)
		return fmt.Errorf("marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"http://"+c.config.HTTP.Address+"/api/bot/notify",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		c.logger.Errorw("Failed to create notification request", "error", err)
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "IoT-Server/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Errorw("Failed to send notification request", "error", err)
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warnw("Bot returned non-OK status",
			"status", resp.StatusCode,
			"user_id", userID,
			"device_id", event.DeviceID,
		)
		return fmt.Errorf("bot status: %d", resp.StatusCode)
	}

	var botResp botResponse
	if err := json.NewDecoder(resp.Body).Decode(&botResp); err != nil {
		c.logger.Errorw("Failed to decode bot response", "error", err)
		return fmt.Errorf("decode response: %w", err)
	}

	if botResp.Status != "ok" {
		return fmt.Errorf("bot returned error status")
	}

	c.logger.Debugw("Notification sent successfully",
		"user_id", userID,
		"device_id", event.DeviceID,
		"event_type", eventType,
	)

	return nil
}
