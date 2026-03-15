package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

func NormalizeDeviceID(deviceID string) string {
	deviceID = strings.TrimPrefix(deviceID, "0x")
	deviceID = strings.TrimPrefix(deviceID, "0X")
	return deviceID
}

func ParseMQTTPayload(payload []byte) (interface{}, error) {
	var generic map[string]interface{}
	if err := json.Unmarshal(payload, &generic); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if _, hasZbReceived := generic["ZbReceived"]; hasZbReceived {
		var msg ZbReceivedMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			return nil, fmt.Errorf("parse ZbReceived: %w", err)
		}
		return msg, nil
	}

	if _, hasZbInfo := generic["ZbInfo"]; hasZbInfo {
		var msg ZbInfoMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			return nil, fmt.Errorf("parse ZbInfo: %w", err)
		}
		return msg, nil
	}

	return nil, fmt.Errorf("unknown message type")
}

func (msg *ZbReceivedMessage) IsMovementEvent() bool {
	for _, event := range msg.ZbReceived {
		if event.Movement == 1 {
			return true
		}
	}
	return false
}

func (msg *ZbReceivedMessage) GetDeviceIDs() []string {
	var ids []string
	for deviceID := range msg.ZbReceived {
		ids = append(ids, NormalizeDeviceID(deviceID))
	}
	return ids
}

func (msg *ZbInfoMessage) GetDeviceIDs() []string {
	var ids []string
	for deviceID := range msg.ZbInfo {
		ids = append(ids, NormalizeDeviceID(deviceID))
	}
	return ids
}

func MapZoneTypeToDeviceType(zoneType int) string {
	if zoneType == 45 {
		return "vibration"
	}
	return "unknown"
}

func DetermineEventType(event ZbDeviceEvent) string {
	if event.Movement == 1 {
		if event.ZoneStatusChange&0x01 != 0 {
			return "alarm"
		}
		return "movement"
	}
	return "status"
}
