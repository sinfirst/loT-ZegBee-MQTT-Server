package models

import "time"

// Device структура соответствует таблице devices
type Device struct {
	DeviceID                 string    `json:"device_id"`
	IEEEAddr                 string    `json:"ieee_addr"`
	UserID                   string    `json:"user_id,omitempty"`
	HubID                    string    `json:"hub_id"`
	ModelID                  string    `json:"model_id"`
	Manufacturer             string    `json:"manufacturer,omitempty"`
	DeviceType               int       `json:"device_type"`
	DeviceStatus             int       `json:"device_status"`
	DeviceOnline             bool      `json:"device_online"`
	BatteryPercentage        int       `json:"battery_percentage"`
	BatteryLastSeenTimestamp time.Time `json:"battery_last_seen_timestamp"`
	LastSeen                 int       `json:"last_seen"` // секунды
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
	RawData     string    `json:"raw_data,omitempty"`
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

type ZbInfoMessage struct {
	ZbInfo map[string]DeviceInfo `json:"ZbInfo"`
}

type ZbReceivedMessage struct {
	ZbReceived map[string]DeviceEvent `json:"ZbReceived"`
}

type DeviceInfo struct {
	Device               string   `json:"Device"`
	IEEEAddr             string   `json:"IEEEAddr"`
	ModelID              string   `json:"ModelId"`
	Manufacturer         string   `json:"Manufacturer"`
	Endpoints            []int    `json:"Endpoints"`
	Config               []string `json:"Config"`
	ZoneType             int      `json:"ZoneType"`
	ZoneStatus           int      `json:"ZoneStatus"`
	Reachable            bool     `json:"Reachable"`
	BatteryPercentage    int      `json:"BatteryPercentage"`
	BatteryLastSeenEpoch int64    `json:"BatteryLastSeenEpoch"`
	LastSeen             int      `json:"LastSeen"`
	LastSeenEpoch        int64    `json:"LastSeenEpoch"`
	LinkQuality          int      `json:"LinkQuality"`
}

type DeviceEvent struct {
	Device               string `json:"Device"`
	HexData              string `json:"0500?00,omitempty"`
	ZoneStatusChange     int    `json:"ZoneStatusChange"`
	ZoneStatusChangeZone int    `json:"ZoneStatusChangeZone"`
	Movement             int    `json:"Movement"`
	Endpoint             int    `json:"Endpoint"`
	LinkQuality          int    `json:"LinkQuality"`
}
