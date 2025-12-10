package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (c *ClientHandlers) PushEvent(event models.Event, userID string) error {
	type botNotification struct {
		UserID     string    `json:"user_id"`
		DeviceID   string    `json:"device_id"`
		Event      string    `json:"event"`
		Confidence float32   `json:"confidence"`
		Timestamp  time.Time `json:"timestamp"`
	}

	type botResponse struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	notification := botNotification{
		UserID:     userID,
		DeviceID:   event.DeviceID,
		Event:      event.Data.Event,
		Confidence: event.Data.EventConfidence,
		Timestamp:  event.Data.TimeStamp,
	}

	jsonData, err := json.Marshal(notification)
	if err != nil {
		c.logger.Error("can't code json notification", err)
		return err
	}

	req, err := http.NewRequest("POST", c.config.HTTP.Address+"/api/bot/notify", bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.Error("failed to create request", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Go-Server/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Errorf("failed to send request: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warnf("server returned status: %d", resp.StatusCode)
	}

	var botResp botResponse
	if err := json.NewDecoder(resp.Body).Decode(&botResp); err != nil {
		c.logger.Errorf("failed to decode response: %v", err)
		return err
	}

	if botResp.Status != "ok" {
		c.logger.Warnf("bot returned error: %s", botResp.Message)
	}

	return nil
}
