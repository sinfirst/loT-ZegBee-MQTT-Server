package models

import "time"

type Device struct {
	DeviceID         string    `json:"device_id"`
	UserID           string    `json:"user_id"`
	HubID            string    `json:"hub_id"`
	DeviceType       string    `json:"device_type"`
	LastEvent        string    `json:"dlast_event"`
	Battery          Battery   `json:"battery"`
	SignalStrength   int       `json:"signal_strength"`
	OrientationState string    `json:"orientation_state"`
	SensorStatus     string    `json:"sensor_status"`
	LastSeen         time.Time `json:"last_seen"`
}

type PushEvent struct {
	UserID     string    `json:"user_id"`
	DeviceID   string    `json:"device_id"`
	Event      string    `json:"event"`
	Confidence string    `json:"confidence"`
	Timestamp  time.Time `json:"timestamp"`
}

type Event struct {
	HubID    string `json:"hub_id"`
	DeviceID string `json:"device_id"`
	Data     Data   `json:"data"`
}

type Data struct {
	Event           string       `json:"event"`
	EventConfidence float32      `json:"event_confidence"`
	Acceleration    Acceleration `json:"acceleration"`
	Angle           Angle        `json:"angle"`
	Battery         Battery      `json:"battery"`
	SignalStrength  int          `json:"signal_strength"`
	Temperature     float32      `json:"temperature"`
	TimeStamp       time.Time    `json:"timestamp"`
}

type Acceleration struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}
type Angle struct {
	Pitch float32 `json:"pitch"`
	Roll  float32 `json:"roll"`
}
type Battery struct {
	Voltage    float32 `json:"voltage"`
	Percentage int     `json:"percentage"`
}
