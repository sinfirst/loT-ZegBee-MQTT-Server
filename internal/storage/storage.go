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
	db, err := pgxpool.New(context.Background(), conf.DataBase.DataBaseDSN)

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
	query := `SELECT 1 FROM pg_database WHERE datname = $1`
	var exists int
	err := db.QueryRow(context.Background(), query, dbName).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *PGD)AddDevices(, devices []Device) ([]string, error) {
    if len(devices) == 0 {
        return []string{}, nil
    }

    deviceIDs := make([]string, 0, len(devices))

    // SQL запрос для вставки данных
    query := `
        INSERT INTO devices (
            device_id, user_id, hub_id, device_type, 
            last_event, battery_type, signal_strength, 
            orientation_state, sensor_status, last_seen
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (device_id) DO UPDATE SET
            user_id = EXCLUDED.user_id,
            hub_id = EXCLUDED.hub_id,
            device_type = EXCLUDED.device_type,
            last_event = EXCLUDED.last_event,
            battery_type = EXCLUDED.battery_type,
            signal_strength = EXCLUDED.signal_strength,
            orientation_state = EXCLUDED.orientation_state,
            sensor_status = EXCLUDED.sensor_status,
            last_seen = EXCLUDED.last_seen,
            updated_at = CURRENT_TIMESTAMP
    `

    // Проходим по всем устройствам и добавляем их
    for _, device := range devices {
        _, err := db.Exec(query,
            device.DeviceID,
            device.UserID,
            device.HubID,
            device.DeviceType,
            device.LastEvent,
            device.Battery.BatteryType,
            device.SignalStrength,
            device.OrientationState,
            device.SensorStatus,
            device.LastSeen,
        )
        
        if err != nil {
            return nil, fmt.Errorf("ошибка при добавлении устройства %s: %w", device.DeviceID, err)
        }
        
        // Добавляем ID устройства в результат
        deviceIDs = append(deviceIDs, device.DeviceID)
    }

    return deviceIDs, nil
}

func (p *PGDB) CreateUser(ctx context.Context, tgID int, username string) (int, error) {
	var id int
	query := `
		INSERT INTO users (telegram_id, username)
		VALUES ($1, $2)
		RETURNING id
	`

	err := p.db.QueryRow(ctx, query, tgID, username).Scan(&id)
	if err != nil {
		p.logger.Errorw("Problem with create in db: ", err)
		return 0, err
	}

	return id, err
}

func (p *PGDB) CreateConnect(ctx context.Context, userID, hubID string) ([]string, error) {
	var deviceIDs []string

	rows, err := p.db.Query(ctx, `
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

	return deviceIDs, nil
}

func (p *PGDB) GetDeviceInfo(ctx context.Context, deviceID string) (models.Device, error) {
	var device models.Device
	
	query := `SELECT device_id, user_id, hub_id, device_type, last_event, battery, signal_strength, orientation_state, sensor_status, last_seen FROM devices WHERE device_id = $1`
	err := p.db.QueryRow(ctx, query, deviceID).Scan(&device.DeviceID, &device.UserID &device.HubID, &device.DeviceType, &device.LastEvent, &device.Battery, &device.SignalStrength, &device.OrientationState, &device.SensorStatus, &device.LastSeen))

	if err != nil {
		p.logger.Errorw("Problem get device info from db: ", err)
		return nil, err
	}

	return device, nil
	
}

func (p *PGDB) DeleteDevice(ctx context.Context, deviceID string) error {
    result, err := p.db.Exec("DELETE FROM devices WHERE device_id = $1", deviceID)
    if err != nil {
        p.logger.Errorw("Problem with delete device from db: ", err)
		return err
    }
    
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


func (p *PGDB) GetEventsByDeviceID(ctx context.Context, deviceID string, hours string) ([]Event, error) {
    startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
    
    query := `
        SELECT 
            id, 
            hub_id, 
            device_id, 
            event_type, 
            event_confidence,
            signal_strength,
            temperature,
            acceleration,
            angle,
            battery,
            timestamp,
        FROM events 
        WHERE device_id = $1 
        AND timestamp >= $2
        ORDER BY timestamp DESC
    `
    
    rows, err := db.Query(query, deviceID, startTime)
    if err != nil {
		p.logger.Errorf("failed to query events: %w", err)
		return nil, err
    }
    defer rows.Close()
    
    var events []Event
    
    for rows.Next() {
        var event Event
        
        err := rows.Scan(
            &event.ID,
            &event.HubID,
            &event.DeviceID,
            &event.EventType,
            &event.EventConfidence,
            &event.SignalStrength,
            &event.Temperature,
            &accelerationData,
            &angleData,
            &batteryData,
            &event.Timestamp,
            &event.CreatedAt,
        )
        
        if err != nil {
			p.logger.Errorf("failed to scan event row: %w", err)
			return nil, err
        }
        
        event.Acceleration = accelerationData
        event.Angle = angleData
        event.Battery = batteryData
        
        events = append(events, event)
    }
    
    return events, nil
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
