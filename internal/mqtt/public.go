package mqtt

import (
	"fmt"
	"time"
)

// SubscribeToHub подписывается на все топики хаба
func (c *MQTTClient) SubscribeToHub(hubID string) error {
	c.subscribedHubs.mutex.Lock()
	defer c.subscribedHubs.mutex.Unlock()

	if c.subscribedHubs.hubs[hubID] {
		c.logger.Debugw("Already subscribed to hub", "hub_id", hubID)
		return nil
	}

	sensorTopic := fmt.Sprintf("tele/%s/+", hubID)
	if token := c.client.Subscribe(sensorTopic, byte(c.config.MQTT.QoS), c.messageHandler); token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe to %s: %w", sensorTopic, token.Error())
	}

	c.subscribedHubs.hubs[hubID] = true
	c.logger.Infow("Subscribed to hub topics",
		"hub_id", hubID,
		"sensor_topic", sensorTopic,
	)

	c.requestZbInfo(hubID)

	return nil
}

// UnsubscribeFromHub отписывается от топиков хаба
func (c *MQTTClient) UnsubscribeFromHub(hubID string) error {
	if !c.subscribedHubs.hubs[hubID] {
		return nil
	}

	sensorTopic := fmt.Sprintf("tele/%s/+/SENSOR", hubID)
	if token := c.client.Unsubscribe(sensorTopic); token.Wait() && token.Error() != nil {
		c.logger.Warnw("Failed to unsubscribe from sensor topic", "hub_id", hubID, "error", token.Error())
	}

	delete(c.subscribedHubs.hubs, hubID)
	c.logger.Infow("Unsubscribed from hub", "hub_id", hubID)
	return nil
}

// RestoreSubscriptions восстанавливает подписки на хабы из БД при старте сервера
func (c *MQTTClient) RestoreSubscriptions(hubs []string) {
	c.logger.Infow("Restoring MQTT subscriptions", "hubs_count", len(hubs))

	for _, hubID := range hubs {
		if err := c.SubscribeToHub(hubID); err != nil {
			c.logger.Errorw("Failed to restore subscription to hub",
				"hub_id", hubID,
				"error", err,
			)

			if c.handlers != nil {
				c.handlers.ErrorHandler(hubID, "", "subscription_restore_failed",
					fmt.Sprintf("Failed to restore MQTT subscription for hub %s", hubID))
			}
		}
	}
}
func (c *MQTTClient) Close() {
	c.logger.Info("Closing MQTT client")

	if c.zbInfoTicker != nil {
		c.zbInfoTicker.Stop()
	}

	for _, hubID := range c.getSubscribedHubs() {
		c.UnsubscribeFromHub(hubID)
		time.Sleep(100 * time.Millisecond)
	}

	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}
}
