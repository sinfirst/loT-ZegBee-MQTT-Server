package notification

import (
	"context"
	"sync"
	"time"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
	"go.uber.org/zap"
)

type storage interface {
	GetRecentEventsByDeviceID(context.Context, string, int) ([]models.Event, error)
	GetDeviceUserPairs(context.Context) ([]string, []string, error)
}

type tgBot interface {
	PushEvent(models.Event, string) error
}

type NotificationPooler struct {
	storage     storage
	tgBot       tgBot
	config      *config.Config
	logger      *zap.SugaredLogger
	stopChan    chan bool
	activePolls map[string]context.CancelFunc // deviceID -> cancel function
	mu          sync.RWMutex
}

func NewNotificationPoolerStruct(storage storage, tgBot tgBot, config *config.Config, logger *zap.SugaredLogger) *NotificationPooler {
	return &NotificationPooler{
		storage:     storage,
		tgBot:       tgBot,
		config:      config,
		logger:      logger,
		stopChan:    make(chan bool),
		activePolls: make(map[string]context.CancelFunc),
	}
}

// StartPollingForDevice запускает пуллер для конкретного устройства
func (n *NotificationPooler) StartPollingForDevice(deviceID string, userID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Проверяем, не запущен ли уже пуллер для этого устройства
	if _, exists := n.activePolls[deviceID]; exists {
		n.logger.Debugw("Polling already active for device", "device_id", deviceID)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.activePolls[deviceID] = cancel

	go n.pollDeviceEvents(ctx, deviceID, userID)
	n.logger.Infow("Started polling for device", "device_id", deviceID, "user_id", userID)
}

// pollDeviceEvents основная логика опроса
func (n *NotificationPooler) pollDeviceEvents(ctx context.Context, deviceID, userID string) {
	ticker := time.NewTicker(time.Duration(n.config.Notify.NotificationTiker) * time.Second)
	defer ticker.Stop()

	n.logger.Debugw("Starting poller for device", "device_id", deviceID)

	for {
		select {
		case <-ticker.C:
			n.checkEvent(deviceID, userID)
		case <-ctx.Done():
			n.logger.Debugw("Poller stopped for device", "device_id", deviceID)
			return
		case <-n.stopChan:
			n.logger.Debugw("Global stop for device poller", "device_id", deviceID)
			return
		}
	}
}

// StopPollingForDevice останавливает пуллер для конкретного устройства
func (n *NotificationPooler) StopPollingForDevice(deviceID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if cancel, exists := n.activePolls[deviceID]; exists {
		cancel()
		delete(n.activePolls, deviceID)
		n.logger.Infow("Stopped polling for device", "device_id", deviceID)
	}
}

// StopAll останавливает все пуллеры
func (n *NotificationPooler) StopAll() {
	n.mu.Lock()
	defer n.mu.Unlock()

	close(n.stopChan)

	for deviceID, cancel := range n.activePolls {
		cancel()
		delete(n.activePolls, deviceID)
		n.logger.Debugw("Stopped polling for device", "device_id", deviceID)
	}

	n.logger.Info("All notification pollers stopped")
}

// RestorePollers восстанавливает пуллеры из БД при старте сервера
func (n *NotificationPooler) RestorePollers() {
	n.logger.Info("Restoring notification pollers from database")

	deviceIDs, userIDs, err := n.storage.GetDeviceUserPairs(context.Background())
	if err != nil {
		n.logger.Errorw("Failed to get device-user pairs for notification restore", "error", err)
		return
	}

	if len(deviceIDs) != len(userIDs) {
		n.logger.Errorw("Device and user arrays mismatch",
			"device_count", len(deviceIDs),
			"user_count", len(userIDs))
		return
	}

	for i := 0; i < len(deviceIDs); i++ {
		if deviceIDs[i] != "" && userIDs[i] != "" {
			n.StartPollingForDevice(deviceIDs[i], userIDs[i])
			n.logger.Debugw("Restored poller for device",
				"device_id", deviceIDs[i],
				"user_id", userIDs[i])
		}
	}

	n.logger.Infow("Notification pollers restored", "devices_count", len(deviceIDs))
}
func (n *NotificationPooler) checkEvent(deviceID string, userID string) {
	events, err := n.storage.GetRecentEventsByDeviceID(context.Background(), deviceID, n.config.Notify.NotificationTiker+15)
	if err != nil {
		n.logger.Errorw("Failed to scan events in poller", "device_id", deviceID, "error", err)
		return
	}
	if len(events) < n.config.Notify.MinEventForNotify {
		return
	}

	firstEvent := events[0]

	linkQualityValues := make([]int, 0, len(events))
	for _, event := range events {
		linkQualityValues = append(linkQualityValues, event.LinkQuality)
	}

	first := linkQualityValues[0]
	last := linkQualityValues[len(linkQualityValues)-1]
	drop := first - last

	if drop > n.config.Notify.DropLinkQuality {
		firstEvent.EventType = "theft_detected"
		n.tgBot.PushEvent(firstEvent, userID)
		return
	}

	maxVariation := 0
	for i := 1; i < len(linkQualityValues); i++ {
		variation := linkQualityValues[i] - linkQualityValues[i-1]
		if variation < 0 {
			variation = -variation
		}
		if variation > maxVariation {
			maxVariation = variation
		}
	}

	if maxVariation > n.config.Notify.MaxVariation {
		n.tgBot.PushEvent(firstEvent, userID)
		return
	}

	firstEvent.EventType = "vibration_detected"
	n.tgBot.PushEvent(firstEvent, userID)
}
