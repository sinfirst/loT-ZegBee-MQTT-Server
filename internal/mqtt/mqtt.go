package mqtt

import (
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
	"go.uber.org/zap"
)

type Handlers interface {
	EventHandler(hubID string, deviceEvent models.ZbDeviceEvent) error
	ZbInfoHandler(hubID string, zbInfo map[string]models.ZbDeviceInfo) error
	ErrorHandler(hubID, deviceID, errorType, message string) error
}

type subscribedHubsMap struct {
	mutex sync.RWMutex
	hubs  map[string]bool
}

type MQTTClient struct {
	client         mqtt.Client
	config         *config.Config
	logger         *zap.SugaredLogger
	handlers       Handlers
	subscribedHubs subscribedHubsMap
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
		subscribedHubs: subscribedHubsMap{hubs: make(map[string]bool)},
		zbInfoTicker:   time.NewTicker(time.Duration(config.MQTT.ZbInfoTiker) * time.Second),
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
