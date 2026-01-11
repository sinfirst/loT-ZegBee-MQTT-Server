package mqtt

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
)

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
	if len(parts) != 4 {
		c.logger.Debugw("Ignoring non-SENSOR topic", "topic", msg.Topic())
		return
	}

	hubID := parts[1]
	deviceID := parts[2]
	topicType := parts[3]

	if topicType != "SENSOR" {
		c.logger.Debugw("Ignoring non-SENSOR message", "topic", msg.Topic())
		return
	}

	parsed, err := models.ParseMQTTPayload(msg.Payload())
	if err != nil {
		c.logger.Errorw("Failed to parse MQTT payload",
			"error", err,
			"topic", msg.Topic(),
		)
		return
	}

	switch v := parsed.(type) {
	case models.ZbReceivedMessage:
		c.handleZbReceivedMessage(hubID, deviceID, v)
	case models.ZbInfoMessage:
		c.handleZbInfoMessage(hubID, v)
	default:
		c.logger.Debugw("Unknown message type", "topic", msg.Topic())
	}

	msg.Ack()
}

func (c *MQTTClient) handleZbReceivedMessage(hubID, deviceID string, msg models.ZbReceivedMessage) {
	if !msg.IsMovementEvent() {
		c.logger.Debugw("Ignoring status message", "hub_id", hubID, "device_id", deviceID)
		return
	}

	normalizedID := models.NormalizeDeviceID(deviceID)

	for msgDeviceID, event := range msg.ZbReceived {
		msgNormalizedID := models.NormalizeDeviceID(msgDeviceID)

		if msgNormalizedID != normalizedID {
			c.logger.Debugw("Device ID mismatch in message",
				"hub_id", hubID,
				"topic_device_id", normalizedID,
				"message_device_id", msgNormalizedID,
			)
			continue
		}

		if err := c.handlers.EventHandler(hubID, event); err != nil {
			c.logger.Errorw("Failed to handle event",
				"hub_id", hubID,
				"device_id", normalizedID,
				"error", err,
			)
		}
		break
	}
}

func (c *MQTTClient) handleZbInfoMessage(hubID string, msg models.ZbInfoMessage) {
	if err := c.handlers.ZbInfoHandler(hubID, msg.ZbInfo); err != nil {
		c.logger.Errorw("Failed to handle ZbInfo",
			"hub_id", hubID,
			"error", err,
		)
	}
}

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

func (c *MQTTClient) getSubscribedHubs() []string {
	c.subscribedHubs.mutex.RLock()
	defer c.subscribedHubs.mutex.RUnlock()

	hubs := make([]string, 0, len(c.subscribedHubs.hubs))
	for hubID := range c.subscribedHubs.hubs {
		hubs = append(hubs, hubID)
	}

	return hubs
}
