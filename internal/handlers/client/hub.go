package client

import (
	"context"
	"fmt"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

// ZbInfoHandler обрабатывает информацию об устройствах хаба
func (c *ClientHandlers) ZbInfoHandler(hubID string, zbInfo map[string]interface{}) error {
	zbInfoData, ok := zbInfo["ZbInfo"].(map[string]interface{})
	if !ok {
		errMsg := "invalid ZbInfo structure"
		c.logger.Errorw(errMsg, "hub_id", hubID, "data", zbInfo)
		return c.ErrorHandler(hubID, "", "zbinfo_parse_error", errMsg)
	}

	deviceIDs := make([]string, 0, len(zbInfoData))
	for deviceID := range zbInfoData {
		deviceIDs = append(deviceIDs, deviceID)
	}

	err := c.storage.UpdateDevicesFromZbInfo(context.Background(), hubID, zbInfoData)
	if err != nil {
		c.logger.Errorw("Failed to update devices from ZbInfo",
			"hub_id", hubID,
			"error", err,
			"device_count", len(zbInfoData),
		)
		return c.ErrorHandler(hubID, "", "zbinfo_update_error",
			fmt.Sprintf("Failed to update %d devices: %v", len(zbInfoData), err))
	}

	if len(deviceIDs) > 0 {
		if err := c.storage.AutoAssignNewDevices(context.Background(), hubID, deviceIDs); err != nil {
			c.logger.Warnw("Failed to auto-assign new devices",
				"hub_id", hubID,
				"error", err,
			)
		}
	}

	c.logger.Infow("Processed ZbInfo",
		"hub_id", hubID,
		"total_devices", len(zbInfoData),
	)

	return nil
}

// EventHandler обрабатывает события от датчиков
func (c *ClientHandlers) EventHandler(hubID, deviceID string, eventData map[string]interface{}) error {
	eventID, err := c.storage.StorageEvent(context.Background(), hubID, deviceID, eventData)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to save event to DB: %v", err)
		c.logger.Errorw(errMsg,
			"hub_id", hubID,
			"device_id", deviceID,
			"error", err,
		)
		return c.ErrorHandler(hubID, deviceID, "db_save_error", errMsg)
	}

	c.logger.Debugw("Event saved to DB",
		"event_id", eventID,
		"hub_id", hubID,
		"device_id", deviceID,
	)

	if c.shouldNotifyUser(eventData) {
		userID, err := c.storage.GetUserIDByDeviceID(context.Background(), deviceID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get user_id for device: %v", err)
			c.logger.Warnw(errMsg,
				"device_id", deviceID,
				"error", err,
			)
			return c.ErrorHandler(hubID, deviceID, "user_lookup_error", errMsg)
		}

		if userID == "" {
			c.logger.Warnw("Device not assigned to any user",
				"hub_id", hubID,
				"device_id", deviceID,
			)
			return nil
		}

		event := models.Event{
			ID:        eventID,
			HubID:     hubID,
			DeviceID:  deviceID,
			EventType: "movement",
		}

		if err := c.pushEvent(event, userID); err != nil {
			errMsg := fmt.Sprintf("Failed to send notification: %v", err)
			c.logger.Errorw(errMsg,
				"user_id", userID,
				"device_id", deviceID,
				"error", err,
			)
			return c.ErrorHandler(hubID, deviceID, "notification_error", errMsg)
		}
	}

	return nil
}

// shouldNotifyUser проверяет, нужно ли отправлять уведомление
func (c *ClientHandlers) shouldNotifyUser(eventData map[string]interface{}) bool {
	zbReceived, ok := eventData["ZbReceived"].(map[string]interface{})
	if !ok {
		return false
	}

	for _, deviceData := range zbReceived {
		if data, ok := deviceData.(map[string]interface{}); ok {
			if movement, ok := data["Movement"].(float64); ok && movement == 1 {
				return true
			}
		}
	}

	return false
}
