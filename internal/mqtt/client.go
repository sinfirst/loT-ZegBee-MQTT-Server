package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/handlers"
	"go.uber.org/zap"
)

type MQTTClient struct {
	client   mqtt.Client
	config   config.Config
	logger   zap.SugaredLogger
	handlers handlers.Handlers
}

func NewMQTTClient(config config.Config, logger zap.SugaredLogger, handlers handlers.Handlers) *MQTTClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.MQTT.Broker)
	opts.SetClientID(config.MQTT.ClientID)

	return &MQTTClient{
		client:   mqtt.NewClient(opts),
		config:   config,
		logger:   logger,
		handlers: handlers,
	}
}

func (c *MQTTClient) Connect() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	c.Subscribe([]string{c.config.MQTT.New_divice_topic})

	return nil
}

func (c *MQTTClient) Subscribe(topics []string) error {
	for _, topic := range topics {
		token := c.client.Subscribe(topic, byte(c.config.MQTT.QoS), c.messageHandler)
		if token.Wait() && token.Error() != nil {
			c.logger.Errorw("failed to subscribe to topic",
				"topic", topic,
				"error", token.Error(),
			)
			return fmt.Errorf("subscribe to %s: %w", topic, token.Error())
		}
		c.logger.Infow("subscribed to topic", "topic", topic)
	}
	return nil
}

func (c *MQTTClient) messageHandler(client mqtt.Client, msg mqtt.Message) {
	c.logger.Debugw("received MQTT message",
		"topic", msg.Topic(),
		"payload_size", len(msg.Payload()),
	)

	if msg.Topic() == c.config.MQTT.New_divice_topic {
		topics := c.handlers.NewDiviceMessageHandler(msg.Payload())
		c.Subscribe(topics)
	} else {
		c.handlers.MessageHandler(msg.Payload())
	}

	msg.Ack()
}
