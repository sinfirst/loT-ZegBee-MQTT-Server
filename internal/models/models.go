package models

import (
	"time"
)

// Device структура соответствует таблице devices
type Device struct {
	DeviceID                 string    `json:"device_id"`
	IEEEAddr                 string    `json:"ieee_addr"`
	UserID                   string    `json:"user_id,omitempty"`
	HubID                    string    `json:"hub_id"`
	ModelID                  string    `json:"model_id"`
	DeviceType               string    `json:"device_type"`
	DeviceStatus             int       `json:"device_status"`
	DeviceOnline             bool      `json:"device_online"`
	BatteryPercentage        int       `json:"battery_percentage"`
	BatteryLastSeenTimestamp time.Time `json:"battery_last_seen_timestamp"`
	LastSeen                 int       `json:"last_seen"`
	LastSeenTimestamp        time.Time `json:"last_seen_timestamp"`
	LinkQuality              int       `json:"link_quality"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// Event структура соответствует таблице events
type Event struct {
	ID          string    `json:"id"`
	HubID       string    `json:"hub_id"`
	DeviceID    string    `json:"device_id"`
	EventType   string    `json:"event_type"`
	LinkQuality int       `json:"link_quality"`
	CreatedAt   time.Time `json:"created_at"`
}

// User структура соответствует таблице users
type User struct {
	ID         string    `json:"id"`
	TelegramID int       `json:"telegram_id"`
	Username   string    `json:"username"`
	HubID      string    `json:"hub_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ZbReceivedMessage представляет сообщение с событиями от устройств
type ZbReceivedMessage struct {
	ZbReceived map[string]ZbDeviceEvent `json:"ZbReceived"`
}

// ZbDeviceEvent содержит данные события от одного устройства
type ZbDeviceEvent struct {
	Device               string `json:"Device"`
	Movement             int    `json:"Movement,omitempty"`
	ZoneStatusChange     int    `json:"ZoneStatusChange,omitempty"`
	ZoneStatusChangeZone int    `json:"ZoneStatusChangeZone,omitempty"`
	HexData              string `json:"0500?00,omitempty"`
	Endpoint             int    `json:"Endpoint,omitempty"`
	LinkQuality          int    `json:"LinkQuality"`
}

// ZbInfoMessage представляет информацию об устройствах хаба
type ZbInfoMessage struct {
	ZbInfo map[string]ZbDeviceInfo `json:"ZbInfo"`
}

// ZbDeviceInfo содержит информацию об устройстве
type ZbDeviceInfo struct {
	Device               string   `json:"Device"`
	IEEEAddr             string   `json:"IEEEAddr"`
	ModelId              string   `json:"ModelId"`
	Manufacturer         string   `json:"Manufacturer,omitempty"`
	Endpoints            []int    `json:"Endpoints,omitempty"`
	Config               []string `json:"Config,omitempty"`
	ZoneType             int      `json:"ZoneType"`
	ZoneStatus           int      `json:"ZoneStatus"`
	Reachable            bool     `json:"Reachable"`
	BatteryPercentage    int      `json:"BatteryPercentage"`
	BatteryLastSeenEpoch int64    `json:"BatteryLastSeenEpoch"`
	LastSeen             int      `json:"LastSeen"`
	LastSeenEpoch        int64    `json:"LastSeenEpoch"`
	LinkQuality          int      `json:"LinkQuality"`
}

