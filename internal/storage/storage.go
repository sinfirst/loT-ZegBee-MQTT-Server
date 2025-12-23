package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
	"go.uber.org/zap"
)

type PGDB struct {
	logger *zap.SugaredLogger
	db     *pgxpool.Pool
}

func NewPGDB(conf *config.Config, logger *zap.SugaredLogger) *PGDB {
	db, err := pgxpool.New(context.Background(), conf.DataBase.DataBaseDSN+"/postgres")

	if err != nil {
		logger.Errorw("Problem with connecting to db: ", err)
		return nil
	}

	err = db.Ping(context.Background())

	if err != nil {
		logger.Errorw("Problem with ping to db: ", err)
		return nil
	}

	exists, err := databaseExists(db, conf.DataBase.Name)
	if err != nil {
		logger.Errorf("failed to check database existence: %w", err)
		return nil
	}

	if !exists {
		_, err = db.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", conf.DataBase.Name))
		if err != nil {
			return nil
		}
	}
	db, err = pgxpool.New(context.Background(), conf.DataBase.DataBaseDSN+"/"+conf.DataBase.Name)

	if err != nil {
		logger.Errorw("Problem with connecting to db: ", err)
		return nil
	}

	return &PGDB{logger: logger, db: db}
}

func databaseExists(db *pgxpool.Pool, dbName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`

	var exists bool
	err := db.QueryRow(context.Background(), query, dbName).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil

}

func (p *PGDB) AddDevices(ctx context.Context, devices []models.Device) ([]string, error) {

	tx, err := p.db.Begin(ctx)
	if err != nil {
		p.logger.Errorw("Problem with create transaction: ", err)
		return nil, err
	}
	defer tx.Rollback(ctx)

	deviceIDs := make([]string, 0, len(devices))

	query := `
        INSERT INTO devices (
            device_id, user_id, hub_id, device_type, 
            last_event, battery_type, signal_strength, 
            orientation_state, sensor_status, last_seen
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `

	for _, device := range devices {
		_, err := tx.Exec(ctx, query,
			device.DeviceID,
			device.UserID,
			device.HubID,
			device.DeviceType,
			device.LastEvent,
			device.Battery,
			device.SignalStrength,
			device.OrientationState,
			device.SensorStatus,
			device.LastSeen,
		)

		if err != nil {
			p.logger.Error("Can't add devices", err)
			return nil, err
		}
		deviceIDs = append(deviceIDs, device.DeviceID)
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.Errorw("Problem with commit transaction: ", err)
		return nil, err
	}

	return deviceIDs, nil
}

func (p *PGDB) StorageEvent(ctx context.Context, event models.Event) (string, error) {
	var id string
	query := `
		INSERT INTO events (hub_id, device_id, event_type, event_confidence, signal_strength, temperature, acceleration, angle, battery, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING device_id
	`
	err := p.db.QueryRow(ctx, query, event.HubID, event.DeviceID, event.Data.Event, event.Data.EventConfidence, event.Data.SignalStrength, event.Data.Temperature, event.Data.Acceleration, event.Data.Angle, event.Data.Battery, event.Data.TimeStamp).Scan(&id)
	if err != nil {
		p.logger.Error("Can't storage event", err)
		return "", err
	}
	return id, nil
}

func (p *PGDB) CreateUser(ctx context.Context, tgID int, username string) (string, error) {
	var id string
	query := `
		INSERT INTO users (telegram_id, username)
		VALUES ($1, $2)
		RETURNING id
	`

	err := p.db.QueryRow(ctx, query, tgID, username).Scan(&id)
	if err != nil {
		p.logger.Errorw("Problem with create in db: ", err)
		return "", err
	}

	return id, err
}

func (p *PGDB) CreateConnect(ctx context.Context, userID, hubID string) ([]string, error) {
	var deviceIDs []string

	tx, err := p.db.Begin(ctx)
	if err != nil {
		p.logger.Errorw("Problem with create transaction: ", err)
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
        UPDATE devices 
        SET user_id = $1
        WHERE hub_id = $2
        RETURNING device_id
    `, userID, hubID)

	if err != nil {
		p.logger.Errorf("failed to update devices: %w", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			p.logger.Errorf("failed to scan device_id: %w", err)
			return nil, err
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.Errorw("Problem with commit transaction: ", err)
		return nil, err
	}

	return deviceIDs, nil
}

func (p *PGDB) GetDeviceInfo(ctx context.Context, deviceID string) (models.Device, error) {
	var device models.Device

	query := `SELECT device_id, user_id, hub_id, device_type, last_event, battery, signal_strength, orientation_state, sensor_status, last_seen FROM devices WHERE device_id = $1`
	err := p.db.QueryRow(ctx, query, deviceID).Scan(&device.DeviceID, &device.UserID, &device.HubID, &device.DeviceType, &device.LastEvent, &device.Battery, &device.SignalStrength, &device.OrientationState, &device.SensorStatus, &device.LastSeen)

	if err != nil {
		p.logger.Errorw("Problem get device info from db: ", err)
		return models.Device{}, err
	}

	return device, nil

}

func (p *PGDB) GetDevicesByUserID(ctx context.Context, userID string) ([]models.Device, error) {
	var devices []models.Device

	query := `SELECT device_id, hub_id, device_type, last_event, battery, signal_strength, orientation_state, sensor_status, last_seen FROM devices WHERE user_id = $1`
	rows, err := p.db.Query(ctx, query, userID)

	if err != nil {
		p.logger.Errorw("Problem with create list devices from db: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var device models.Device

		err := rows.Scan(&device.DeviceID, &device.HubID, &device.DeviceType, &device.LastEvent, &device.Battery, &device.SignalStrength, &device.OrientationState, &device.SensorStatus, &device.LastSeen)
		if err != nil {
			p.logger.Errorw("Problem with create list devices from db: ", err)
			return nil, err
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func (p *PGDB) GetEventsByUserID(ctx context.Context, userID, hours string) ([]models.Event, error) {
	query := `
        SELECT 
            e.hub_id,
            e.device_id,
            e.event_type,
            e.event_confidence,
            e.signal_strength,
            e.temperature,
            ROW(e.acceleration.x, e.acceleration.y, e.acceleration.z)::acceleration_type as acceleration,
            ROW(e.angle.pitch, e.angle.roll)::angle_type as angle,
            ROW(e.battery.voltage, e.battery.percentage)::battery_type as battery,
            e.timestamp
        FROM events e
        INNER JOIN devices d ON e.device_id = d.device_id
        WHERE d.user_id = $1 
            AND e.timestamp >= NOW() - ($2 || ' hours')::INTERVAL
        ORDER BY e.timestamp DESC
    `

	rows, err := p.db.Query(ctx, query, userID, hours)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by user_id: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var acceleration models.Acceleration
		var angle models.Angle
		var battery models.Battery

		err := rows.Scan(
			&event.HubID,
			&event.DeviceID,
			&event.Data.Event,
			&event.Data.EventConfidence,
			&event.Data.SignalStrength,
			&event.Data.Temperature,
			&acceleration,
			&angle,
			&battery,
			&event.Data.TimeStamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		event.Data.Acceleration = acceleration
		event.Data.Angle = angle
		event.Data.Battery = battery
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, nil
}

func (p *PGDB) GetEventsByDeviceID(ctx context.Context, deviceID, hours string) ([]models.Event, error) {
	query := `
        SELECT 
            e.hub_id,
            e.device_id,
            e.event_type,
            e.event_confidence,
            e.signal_strength,
            e.temperature,
            ROW(e.acceleration.x, e.acceleration.y, e.acceleration.z)::acceleration_type as acceleration,
            ROW(e.angle.pitch, e.angle.roll)::angle_type as angle,
            ROW(e.battery.voltage, e.battery.percentage)::battery_type as battery,
            e.timestamp
        FROM events e
        WHERE e.device_id = $1 
            AND e.timestamp >= NOW() - ($2 || ' hours')::INTERVAL
        ORDER BY e.timestamp DESC
    `

	rows, err := p.db.Query(ctx, query, deviceID, hours)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by device_id: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var acceleration models.Acceleration
		var angle models.Angle
		var battery models.Battery

		err := rows.Scan(
			&event.HubID,
			&event.DeviceID,
			&event.Data.Event,
			&event.Data.EventConfidence,
			&event.Data.SignalStrength,
			&event.Data.Temperature,
			&acceleration,
			&angle,
			&battery,
			&event.Data.TimeStamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		event.Data.Acceleration = acceleration
		event.Data.Angle = angle
		event.Data.Battery = battery
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, nil
}

func (p *PGDB) DeleteDevice(ctx context.Context, deviceID string) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		p.logger.Errorw("Problem with create transaction: ", err)
		return err
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM devices WHERE device_id = $1`

	if _, err := tx.Exec(ctx, query, deviceID); err != nil {
		p.logger.Errorw("failed to delete device: %w", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.Errorw("Problem with commit transaction: ", err)
		return err
	}
	return nil
}

func (p *PGDB) GetUserIDByDeviceID(ctx context.Context, deviceID string) (string, error) {
	var id string
	query := `SELECT user_id FROM devices WHERE device_id = $1`
	err := p.db.QueryRow(ctx, query, deviceID).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil

}

func (p *PGDB) ConnectExistByHubID(ctx context.Context, hubID string) (bool, error) {
	var occupiedCount int

	query := `
        SELECT COUNT(*) 
        FROM devices 
        WHERE hub_id = $1 
        AND user_id IS NOT NULL 
        AND user_id != ''
    `

	err := p.db.QueryRow(ctx, query, hubID).Scan(&occupiedCount)
	if err != nil {
		return false, err
	}

	return occupiedCount != 0, nil
}
func (p *PGDB) UserExistsByTGID(ctx context.Context, userID int) (bool, error) {
	var exists bool
	err := p.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE telegram_id = $1
		)
	`, userID).Scan(&exists)
	if err != nil {
		p.logger.Errorw("Problem with check user exist by tg id: ", err)
		return false, err
	}
	return exists, err
}

func (p *PGDB) UserExistsByUserID(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := p.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE id = $1
		)
	`, userID).Scan(&exists)
	if err != nil {
		p.logger.Errorw("Problem check user exist by user id: ", err)
		return false, err
	}
	return exists, err
}

func (p *PGDB) DeviceExistByDeviceID(ctx context.Context, deviceID string) (bool, error) {
	var exists bool
	err := p.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM devices WHERE device_id = $1
		)
	`, deviceID).Scan(&exists)
	if err != nil {
		p.logger.Errorw("Problem check user exist by user id: ", err)
		return false, err
	}
	return exists, err
}
func InitMigrations(conf *config.Config) error {
	db, err := sql.Open("pgx", conf.DataBase.DataBaseDSN+"/"+conf.DataBase.Name)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.Up(db, "internal/storage/migrations"); err != nil {
		return err
	}

	return nil
}
