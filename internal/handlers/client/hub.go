package client

import (
	"context"
	"fmt"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (c *ClientHandlers) ZbInfoHandler(hubID string, zbInfo map[string]models.ZbDeviceInfo) error {
	if len(zbInfo) == 0 {
		return c.ErrorHandler(hubID, "", "empty_zbinfo", "Received empty ZbInfo")
	}

	deviceIDs := make([]string, 0, len(zbInfo))
	for deviceID := range zbInfo {
		deviceIDs = append(deviceIDs, models.NormalizeDeviceID(deviceID))
	}

	err := c.storage.UpdateDevicesFromZbInfo(context.Background(), hubID, zbInfo)
	if err != nil {
		c.logger.Errorw("Failed to update devices from ZbInfo",
			"hub_id", hubID,
			"error", err,
			"device_count", len(zbInfo),
		)
		return c.ErrorHandler(hubID, "", "zbinfo_update_error",
			fmt.Sprintf("Failed to update %d devices: %v", len(zbInfo), err))
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
		"total_devices", len(zbInfo),
	)

	return nil
}

func (c *ClientHandlers) EventHandler(hubID string, eventData models.ZbDeviceEvent) error {
	deviceID := models.NormalizeDeviceID(eventData.Device)

	eventType := determineEventTypeLogic(eventData)

	eventID, err := c.storage.StorageEvent(context.Background(), hubID, deviceID, eventData, eventType)
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
		"event_type", eventType,
	)

	if eventData.Movement == 1 {
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
			EventType: eventType,
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

func determineEventTypeLogic(event models.ZbDeviceEvent) string {
	if event.Movement != 1 {
		return "status"
	}

	if event.ZoneStatusChange&0x01 != 0 {
		return "alarm"
	}

	if event.ZoneStatusChange&0x08 != 0 {
		return "tamper"
	}

	if event.ZoneStatusChange&0x10 != 0 {
		return "low_battery"
	}

	if event.LinkQuality < 50 {
		return "weak_signal"
	}

	return "movement"
}
