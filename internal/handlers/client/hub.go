package handlers

import (
	"encoding/json"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

func (c *ClientHandlers) NewDeviceMessageHandler(msg []byte) []string {
	type newDeivices struct {
		HubID   string          `json:"hub_id"`
		Devices []models.Device `json:"devices"`
	}

	var devices newDeivices
	if err := json.Unmarshal(msg, &devices); err != nil {
		c.logger.Error("Can't decode event", err)
		return nil
	}

	devicesID := c.storage.AddDevices(devices.Devices, devices.HubID)

	return devicesID

}

func (c *ClientHandlers) EventHandler(msg []byte) {
	var event models.Event
	if err := json.Unmarshal(msg, &event); err != nil {
		c.logger.Error("Can't decode event", err)
		return
	}

	userID, err := c.storage.StorageEvent(event)
	if err != nil {
		c.logger.Error("Can't storage in bd", err)
	}

	//обработчик принятия решения
	if check := c.handler(event); check {
		c.PushEvent(event, userID)
	}

}

func (c *ClientHandlers) handler(event models.Event) bool {
	// некая логика принятия решения об отправке пуша
	return true
}
