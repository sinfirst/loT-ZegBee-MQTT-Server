package handlers

import (
	"context"
	"encoding/json"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (c *ClientHandlers) NewDeviceMessageHandler(msg []byte) []string {
	type newDeivices struct {
		Devices []models.Device `json:"devices"`
	}

	var devices newDeivices
	if err := json.Unmarshal(msg, &devices); err != nil {
		c.logger.Error("Can't decode event", err)
		return nil
	}

	devicesID, err := c.storage.AddDevices(context.Background(), devices.Devices)

	if err != nil {
		c.logger.Error("Can't add devices", err)
		return nil
	}

	return devicesID

}

func (c *ClientHandlers) EventHandler(msg []byte) {
	var event models.Event
	if err := json.Unmarshal(msg, &event); err != nil {
		c.logger.Error("Can't decode event", err)
		return
	}

	deviceID, err := c.storage.StorageEvent(context.Background(), event)
	if err != nil {
		c.logger.Error("Can't storage in bd", err)
		return
	}

	//обработчик принятия решения
	if check := c.handler(event); check {
		userID, err := c.storage.GetUserIDByDeviceID(context.Background(), deviceID)
		if err != nil {
			c.logger.Error("Can't storage in bd", err)
			return
		}
		c.PushEvent(event, userID)
	}

}

func (c *ClientHandlers) handler(event models.Event) bool {
	// некая логика принятия решения об отправке пуша
	return true
}
