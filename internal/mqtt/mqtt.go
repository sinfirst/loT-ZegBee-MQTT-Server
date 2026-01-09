package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"go.uber.org/zap"
)

type Handlers interface {
	EventHandler(hubID, deviceID string, eventData map[string]interface{}) error
	ZbInfoHandler(hubID string, zbInfo map[string]interface{}) error
	ErrorHandler(hubID, deviceID, errorType, message string) error
}

type MQTTClient struct {
	client         mqtt.Client
	config         *config.Config
	logger         *zap.SugaredLogger
	handlers       Handlers
	subscribedHubs map[string]bool
	hubMu          sync.RWMutex
	zbInfoTicker   *time.Ticker
	stopTicker     chan bool
}

func NewMQTTClient(config *config.Config, logger *zap.SugaredLogger, handlers Handlers) (*MQTTClient, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.MQTT.Broker)
	opts.SetClientID(config.MQTT.ClientID)

	if config.MQTT.Username != "" {
		opts.SetUsername(config.MQTT.Username)
		opts.SetPassword(config.MQTT.Password)
	}

	opts.SetCleanSession(config.MQTT.CleanSession)
	opts.SetKeepAlive(time.Duration(config.MQTT.KeepAlive) * time.Second)
	opts.SetConnectTimeout(time.Duration(config.MQTT.ConnectTimeout) * time.Second)

	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)

	mqttClient := &MQTTClient{
		config:         config,
		logger:         logger,
		handlers:       handlers,
		subscribedHubs: make(map[string]bool),
		zbInfoTicker:   time.NewTicker(60 * time.Second),
		stopTicker:     make(chan bool),
	}

	opts.OnConnect = func(c mqtt.Client) {
		logger.Info("MQTT client connected", "broker", config.MQTT.Broker)

		go func() {
			time.Sleep(1 * time.Second)
			mqttClient.resubscribeToAllHubs()
		}()
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		logger.Warnw("MQTT connection lost", "error", err)
	}

	opts.OnReconnecting = func(c mqtt.Client, co *mqtt.ClientOptions) {
		logger.Info("MQTT client reconnecting...")
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("mqtt connect: %w", token.Error())
	}

	mqttClient.client = client

	go mqttClient.startZbInfoPoller()

	return mqttClient, nil
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

// SubscribeToHub подписывается на все топики хаба
func (c *MQTTClient) SubscribeToHub(hubID string) error {
	c.hubMu.Lock()
	defer c.hubMu.Unlock()

	if c.subscribedHubs[hubID] {
		c.logger.Debugw("Already subscribed to hub", "hub_id", hubID)
		return nil
	}

	sensorTopic := fmt.Sprintf("tele/%s/+/SENSOR", hubID)
	if token := c.client.Subscribe(sensorTopic, byte(c.config.MQTT.QoS), c.messageHandler); token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe to %s: %w", sensorTopic, token.Error())
	}

	c.subscribedHubs[hubID] = true
	c.logger.Infow("Subscribed to hub topics",
		"hub_id", hubID,
		"sensor_topic", sensorTopic,
	)

	c.requestZbInfo(hubID)

	return nil
}

// resubscribeToAllHubs переподписывается на все хабы при восстановлении соединения
func (c *MQTTClient) resubscribeToAllHubs() {
	c.hubMu.RLock()
	hubs := make([]string, 0, len(c.subscribedHubs))
	for hubID := range c.subscribedHubs {
		hubs = append(hubs, hubID)
	}
	c.hubMu.RUnlock()

	for _, hubID := range hubs {
		c.hubMu.Lock()
		delete(c.subscribedHubs, hubID)
		c.hubMu.Unlock()

		if err := c.SubscribeToHub(hubID); err != nil {
			c.logger.Errorw("Failed to resubscribe to hub",
				"hub_id", hubID,
				"error", err,
			)
		}
	}
}

// UnsubscribeFromHub отписывается от топиков хаба
func (c *MQTTClient) UnsubscribeFromHub(hubID string) error {
	if !c.subscribedHubs[hubID] {
		return nil
	}

	sensorTopic := fmt.Sprintf("tele/%s/+/SENSOR", hubID)
	if token := c.client.Unsubscribe(sensorTopic); token.Wait() && token.Error() != nil {
		c.logger.Warnw("Failed to unsubscribe from sensor topic", "hub_id", hubID, "error", token.Error())
	}

	delete(c.subscribedHubs, hubID)
	c.logger.Infow("Unsubscribed from hub", "hub_id", hubID)
	return nil
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

// getSubscribedHubs возвращает список подписанных хабов
func (c *MQTTClient) getSubscribedHubs() []string {
	c.hubMu.RLock()
	defer c.hubMu.RUnlock()

	hubs := make([]string, 0, len(c.subscribedHubs))
	for hubID := range c.subscribedHubs {
		hubs = append(hubs, hubID)
	}

	return hubs
}

// startZbInfoPoller периодически опрашивает все подписанные хабы
func (c *MQTTClient) startZbInfoPoller() {
	defer c.zbInfoTicker.Stop()

	for {
		select {
		case <-c.zbInfoTicker.C:
			c.logger.Debug("Running ZbInfo poller")

			hubs := c.getSubscribedHubs()
			for _, hubID := range hubs {
				c.requestZbInfo(hubID)
				time.Sleep(100 * time.Millisecond)
			}

		case <-c.stopTicker:
			c.logger.Info("ZbInfo poller stopped")
			return
		}
	}
}
func (c *MQTTClient) Close() {
	c.logger.Info("Closing MQTT client")

	if c.zbInfoTicker != nil {
		c.zbInfoTicker.Stop()
	}

	for hubID := range c.subscribedHubs {
		c.UnsubscribeFromHub(hubID)
	}

	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}
}
