package mqtt

import (
	"fmt"
	"log"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
)

type MQTTClient struct {
	config config.Config
	client MQTT.Client
}

func NewMQTTClient(cfg config.Config) *MQTTClient {
	return &MQTTClient{config: cfg}
}

func (c *MQTTClient) Connect() error {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(c.config.MQTT.Broker)
	opts.SetClientID(fmt.Sprintf(c.config.MQTT.New_divice_topic))
	opts.SetUsername(c.config.MQTT.Username)
	opts.SetPassword(c.config.MQTT.Password)
	opts.KeepAlive(c.config.MQTT.KeepAlive * int(time.Second))
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetOnConnectHandler(c.onConnect)
	opts.SetConnectionLostHandler(c.onConnectionLost)

	c.client = MQTT.NewClient(opts)

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func (c *MQTTClient) Subscribe(topic string) error {
	token := c.client.Subscribe(topic, c.config.MQTT.QoS, handlers.MessageDataHandler())

	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	log.Printf("✅ Подписались на топик: %s", topic)
	return nil
}

func (c *MQTTClient) Disconnect() {
	c.client.Disconnect(250)
	log.Println("Отключились от MQTT брокера")
}

func (c *MQTTClient) onConnect(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

func (c *MQTTClient) onConnectionLost(client mqtt.Client, err error) {
	log.Printf("Connection to MQTT broker lost: %v", err)
}
