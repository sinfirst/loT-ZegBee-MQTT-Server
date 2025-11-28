package models

// type NewDataFromDevice struct {
// 	DeviceID        string       `json:"device_id"`
// 	Event           string       `json:"event"`
// 	EventConfidence float32      `json:"event_confidence"`
// 	Acceleration    Acceleration `json:"acceleration"`
// 	Angle           Angle        `json:"angle"`
// 	Battery         Battery      `json:"battery"`
// 	SignalStrength  int          `json:"signal_strength"`
// 	TimeStamp       string       `json:"timestamp"`
// 	Version         string       `json:"firmware_version"`
// }
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

type RegisterNewDevices struct {
	Devices  []Device `json:"devices"`
	SomeMeta string   `json:"some_meta,omitempty"`
}

type Device struct {
	DeviceID         string  `json:"device_id"`
	DeviceType       string  `json:"device_type"`
	LastEvent        string  `json:"dlast_event"`
	Battery          Battery `json:"battery"`
	SignalStrength   int     `json:"signal_strength"`
	OrientationState string  `json:"orientation_state"`
	SensorStatus     string  `json:"sensor_status"`
	LastSeen         string  `json:"last_seen"`
}

type Event struct {
	DeviceID   string  `json:"device_id"`
	Event      string  `json:"event"`
	Confidence float32 `json:"confidence"`
	TimeStamp  string  `json:"timestamp"`
}
