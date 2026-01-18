package notification

import (
	"context"
	"time"

	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
	"go.uber.org/zap"
)

type storage interface {
	GetRecentEventsByDeviceID(context.Context, string, int) ([]models.Event, error)
}

type tgBot interface {
	PushEvent(models.Event, string) error
}
type NotificationPooler struct {
	storage    storage
	tgBot      tgBot
	config     *config.Config
	ticker     *time.Ticker
	logger     *zap.SugaredLogger
	stopTicker chan bool
}

func NewNotificationPoolerStruct(storage storage, tgBot tgBot, config *config.Config, logger *zap.SugaredLogger) *NotificationPooler {
	return &NotificationPooler{
		storage: storage,
		tgBot:   tgBot,
		config:  config,
		ticker:  time.NewTicker(time.Duration(config.Notify.NotificationTiker) * time.Second),
		logger:  logger,
	}
}

func (n *NotificationPooler) StartPooler(deviceID string, userID string) {
	defer n.ticker.Stop()

	for {
		select {
		case <-n.ticker.C:
			n.logger.Debug("Running check event poller")
			n.checkEvent(deviceID, userID)

		case <-n.stopTicker:
			n.logger.Info("Check event poller stopped")
			return
		}
	}
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

func (n *NotificationPooler) Stop() {
	n.ticker.Stop()
}
