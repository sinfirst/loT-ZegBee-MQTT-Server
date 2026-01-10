package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// resubscribeToAllHubs переподписывается на все хабы при восстановлении соединения
func (c *MQTTClient) resubscribeToAllHubs() {
	for _, hubID := range c.getSubscribedHubs() {
		if err := c.SubscribeToHub(hubID); err != nil {
			c.logger.Errorw("Failed to resubscribe to hub",
				"hub_id", hubID,
				"error", err,
			)
		}
	}
}
func (c *MQTTClient) messageHandler(client mqtt.Client, msg mqtt.Message) {
	c.logger.Debugw("Received MQTT message",
		"topic", msg.Topic(),
		"qos", msg.Qos(),
		"payload_size", len(msg.Payload()),
	)

	parts := strings.Split(msg.Topic(), "/")
	if len(parts) < 4 {
		c.logger.Warnw("Invalid topic format", "topic", msg.Topic())
		return
	}

	hubID := parts[1]
	deviceID := parts[2]
	topicType := parts[3]

	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		c.logger.Errorw("Failed to unmarshal MQTT payload", "error", err, "topic", msg.Topic())
		return
	}

	switch topicType {
	case "SENSOR":
		c.handleSensorMessage(hubID, deviceID, payload)
	default:
		c.logger.Debugw("Unknown topic type", "type", topicType, "topic", msg.Topic())
	}

	msg.Ack()
}

func (c *MQTTClient) handleSensorMessage(hubID, deviceID string, payload map[string]interface{}) {
	if _, ok := payload["ZbReceived"]; ok {
		if err := c.handlers.EventHandler(hubID, deviceID, payload); err != nil {
			c.logger.Errorw("Failed to handle event",
				"hub_id", hubID,
				"device_id", deviceID,
				"error", err,
			)
		}
	} else if _, ok := payload["ZbInfo"]; ok {
		if err := c.handlers.ZbInfoHandler(hubID, payload); err != nil {
			c.logger.Errorw("Failed to handle ZbInfo",
				"hub_id", hubID,
				"error", err,
			)
		}
	} else {
		c.logger.Debugw("Unknown sensor message type", "hub_id", hubID, "device_id", deviceID)
	}
}

// Запрос информации об устройствах хаба
func (c *MQTTClient) requestZbInfo(hubID string) {
	topic := fmt.Sprintf("cmnd/%s/ZbInfo", hubID)
	payload := ""

	if token := c.client.Publish(topic, byte(c.config.MQTT.QoS), false, payload); token.Wait() && token.Error() != nil {
		c.logger.Errorw("Failed to publish ZbInfo request",
			"hub_id", hubID,
			"error", token.Error(),
		)
	} else {
		c.logger.Debugw("Sent ZbInfo request", "hub_id", hubID, "topic", topic)
	}
}

// startZbInfoPoller периодически опрашивает все подписанные хабы
func (c *MQTTClient) startZbInfoPoller() {
	defer c.zbInfoTicker.Stop()

	for {
		select {
		case <-c.zbInfoTicker.C:
			c.logger.Debug("Running ZbInfo poller")

			for _, hubID := range c.getSubscribedHubs() {
				c.requestZbInfo(hubID)
				time.Sleep(100 * time.Millisecond)
			}

		case <-c.stopTicker:
			c.logger.Info("ZbInfo poller stopped")
			return
		}
	}
}

// getSubscribedHubs возвращает список подписанных хабов
func (c *MQTTClient) getSubscribedHubs() []string {
	c.subscribedHubs.mutex.RLock()
	defer c.subscribedHubs.mutex.RUnlock()

	hubs := make([]string, 0, len(c.subscribedHubs.hubs))
	for hubID := range c.subscribedHubs.hubs {
		hubs = append(hubs, hubID)
	}

	return hubs
}
