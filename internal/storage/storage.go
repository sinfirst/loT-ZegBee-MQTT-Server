package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
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

// ========== USER METHODS ==========

func (p *PGDB) CreateUser(ctx context.Context, tgID int, username string) (string, error) {
	var id string
	query := `INSERT INTO users (telegram_id, username) VALUES ($1, $2) RETURNING id::text`
	err := p.db.QueryRow(ctx, query, tgID, username).Scan(&id)
	if err != nil {
		p.logger.Errorw("Failed to create user", "error", err, "tg_id", tgID)
		return "", fmt.Errorf("create user: %w", err)
	}
	return id, nil
}

func (p *PGDB) UserExistsByTGID(ctx context.Context, tgID int) (bool, string, error) {
	var userID sql.NullString

	query := `
		WITH user_check AS (
			SELECT id::text, EXISTS(SELECT 1 FROM users WHERE telegram_id = $1) as exists_flag
			FROM users 
			WHERE telegram_id = $1
		)
		SELECT 
			COALESCE(exists_flag, false),
			id
		FROM user_check
		UNION ALL
		SELECT false, NULL
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE telegram_id = $1)
		LIMIT 1
	`

	var exists bool
	err := p.db.QueryRow(ctx, query, tgID).Scan(&exists, &userID)
	if err != nil {
		p.logger.Errorw("Failed to check user by tg_id",
			"telegram_id", tgID,
			"error", err,
		)
		return false, "", fmt.Errorf("check user by tg_id: %w", err)
	}

	if exists && userID.Valid {
		return true, userID.String, nil
	}

	return false, "", nil
}

func (p *PGDB) UserExistsByUserID(ctx context.Context, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err := p.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		p.logger.Errorw("Failed to check user by user_id", "error", err)
		return false, fmt.Errorf("check user by user_id: %w", err)
	}
	return exists, nil
}

func (p *PGDB) GetUserIDByDeviceID(ctx context.Context, deviceID string) (string, error) {
	var userID sql.NullString
	query := `SELECT user_id::text FROM devices WHERE device_id = $1`
	err := p.db.QueryRow(ctx, query, deviceID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("device not found: %s", deviceID)
		}
		p.logger.Errorw("Failed to get user_id by device_id", "error", err)
		return "", fmt.Errorf("get user by device: %w", err)
	}

	if !userID.Valid || userID.String == "" {
		return "", fmt.Errorf("device not assigned to any user")
	}

	return userID.String, nil
}

// ========== DEVICE METHODS ==========

func (p *PGDB) CreateConnect(ctx context.Context, userID, hubID string) ([]string, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		p.logger.Errorw("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	updateQuery := `UPDATE devices SET user_id = $1 WHERE hub_id = $2 AND user_id IS NULL RETURNING device_id`
	rows, err := tx.Query(ctx, updateQuery, userID, hubID)
	if err != nil {
		p.logger.Errorw("Failed to update devices", "error", err)
		return nil, fmt.Errorf("update devices: %w", err)
	}
	defer rows.Close()

	var deviceIDs []string
	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			p.logger.Errorw("Failed to scan device_id", "error", err)
			return nil, fmt.Errorf("scan device_id: %w", err)
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	if err := rows.Err(); err != nil {
		p.logger.Errorw("Rows iteration error", "error", err)
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	updateUserQuery := `UPDATE users SET hub_id = $1 WHERE id = $2`
	_, err = tx.Exec(ctx, updateUserQuery, hubID, userID)
	if err != nil {
		p.logger.Errorw("Failed to update user hub_id", "error", err)
		return nil, fmt.Errorf("update user hub: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.Errorw("Failed to commit transaction", "error", err)
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return deviceIDs, nil
}

func (p *PGDB) ConnectExistByHubID(ctx context.Context, hubID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE hub_id = $1 AND user_id IS NOT NULL)`
	err := p.db.QueryRow(ctx, query, hubID).Scan(&exists)
	if err != nil {
		p.logger.Errorw("Failed to check hub connection", "error", err)
		return false, fmt.Errorf("check hub connection: %w", err)
	}
	return exists, nil
}

func (p *PGDB) GetDevicesByUserID(ctx context.Context, userID string) ([]models.Device, error) {
	query := `
		SELECT 
			device_id, 
			ieee_addr, 
			user_id::text, 
			hub_id, 
			model_id, 
			device_type, 
			device_status, 
			device_online, 
			battery_percentage, 
			battery_last_seen_timestamp, 
			last_seen, 
			last_seen_timestamp, 
			link_quality,
			created_at,
			updated_at
		FROM devices 
		WHERE user_id = $1
		ORDER BY last_seen_timestamp DESC
	`

	rows, err := p.db.Query(ctx, query, userID)
	if err != nil {
		p.logger.Errorw("Failed to get devices by user_id", "error", err)
		return nil, fmt.Errorf("get devices by user: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var device models.Device
		err := rows.Scan(
			&device.DeviceID,
			&device.IEEEAddr,
			&device.UserID,
			&device.HubID,
			&device.ModelID,
			&device.DeviceType,
			&device.DeviceStatus,
			&device.DeviceOnline,
			&device.BatteryPercentage,
			&device.BatteryLastSeenTimestamp,
			&device.LastSeen,
			&device.LastSeenTimestamp,
			&device.LinkQuality,
			&device.CreatedAt,
			&device.UpdatedAt,
		)
		if err != nil {
			p.logger.Errorw("Failed to scan device", "error", err)
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

func (p *PGDB) GetDeviceInfo(ctx context.Context, deviceID string) (models.Device, error) {
	var device models.Device
	query := `
		SELECT 
			device_id, 
			ieee_addr, 
			user_id::text, 
			hub_id, 
			model_id, 
			device_type, 
			device_status, 
			device_online, 
			battery_percentage, 
			battery_last_seen_timestamp, 
			last_seen, 
			last_seen_timestamp, 
			link_quality,
			created_at,
			updated_at
		FROM devices 
		WHERE device_id = $1
	`

	err := p.db.QueryRow(ctx, query, deviceID).Scan(
		&device.DeviceID,
		&device.IEEEAddr,
		&device.UserID,
		&device.HubID,
		&device.ModelID,
		&device.DeviceType,
		&device.DeviceStatus,
		&device.DeviceOnline,
		&device.BatteryPercentage,
		&device.BatteryLastSeenTimestamp,
		&device.LastSeen,
		&device.LastSeenTimestamp,
		&device.LinkQuality,
		&device.CreatedAt,
		&device.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Device{}, fmt.Errorf("device not found: %s", deviceID)
		}
		p.logger.Errorw("Failed to get device info", "error", err)
		return models.Device{}, fmt.Errorf("get device info: %w", err)
	}

	return device, nil
}

func (p *PGDB) DeviceExistByDeviceID(ctx context.Context, deviceID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE device_id = $1)`
	err := p.db.QueryRow(ctx, query, deviceID).Scan(&exists)
	if err != nil {
		p.logger.Errorw("Failed to check device existence", "error", err)
		return false, fmt.Errorf("check device existence: %w", err)
	}
	return exists, nil
}

// UpdateDevicesFromZbInfo обновляет или создает все устройства из ZbInfo
func (p *PGDB) UpdateDevicesFromZbInfo(ctx context.Context, hubID string, zbInfo map[string]interface{}) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		p.logger.Errorw("Failed to begin transaction for ZbInfo update",
			"hub_id", hubID,
			"error", err,
		)
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for deviceID, deviceData := range zbInfo {
		deviceInfo, ok := deviceData.(map[string]interface{})
		if !ok {
			p.logger.Warnw("Invalid device data in ZbInfo",
				"device_id", deviceID,
				"hub_id", hubID,
			)
			continue
		}

		ieeeAddr, _ := deviceInfo["IEEEAddr"].(string)
		modelID, _ := deviceInfo["ModelId"].(string)
		deviceType, _ := deviceInfo["ZoneType"].(float64)
		battery, _ := deviceInfo["BatteryPercentage"].(float64)
		lastSeen, _ := deviceInfo["LastSeen"].(float64)
		linkQuality, _ := deviceInfo["LinkQuality"].(float64)
		reachable, _ := deviceInfo["Reachable"].(bool)

		deviceStatus := 0
		if reachable {
			deviceStatus = 1
		}

		query := `
			INSERT INTO devices (
				device_id, ieee_addr, hub_id, model_id,
				device_type, device_status, device_online, 
				battery_percentage, last_seen, link_quality
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (device_id) DO UPDATE SET
				ieee_addr = EXCLUDED.ieee_addr,
				hub_id = EXCLUDED.hub_id,
				model_id = EXCLUDED.model_id,
				device_type = EXCLUDED.device_type,
				device_status = EXCLUDED.device_status,
				device_online = EXCLUDED.device_online,
				battery_percentage = EXCLUDED.battery_percentage,
				last_seen = EXCLUDED.last_seen,
				link_quality = EXCLUDED.link_quality,
				updated_at = CURRENT_TIMESTAMP
		`

		_, err := tx.Exec(ctx, query,
			deviceID,
			ieeeAddr,
			hubID,
			modelID,
			deviceType,
			deviceStatus,
			reachable,
			int(battery),
			int(lastSeen),
			int(linkQuality),
		)

		if err != nil {
			p.logger.Warnw("Failed to upsert device from ZbInfo",
				"device_id", deviceID,
				"hub_id", hubID,
				"error", err,
			)
			break
		}

	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.Errorw("Failed to commit ZbInfo transaction",
			"hub_id", hubID,
			"error", err,
		)
		return fmt.Errorf("commit transaction: %w", err)
	}

	p.logger.Debugw("Updated devices from ZbInfo in single transaction",
		"hub_id", hubID,
		"total", len(zbInfo),
	)

	return nil
}

func (p *PGDB) DeleteDevice(ctx context.Context, deviceID string) error {
	query := `DELETE FROM devices WHERE device_id = $1`
	result, err := p.db.Exec(ctx, query, deviceID)
	if err != nil {
		p.logger.Errorw("Failed to delete device", "error", err)
		return fmt.Errorf("delete device: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	return nil
}

// ========== EVENT METHODS ==========

func (p *PGDB) StorageEvent(ctx context.Context, hubID, deviceID string, eventData map[string]interface{}) (string, error) {
	zbReceived, ok := eventData["ZbReceived"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid ZbReceived structure")
	}

	deviceData, ok := zbReceived[deviceID].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("device %s not found in ZbReceived", deviceID)
	}

	movement, _ := deviceData["Movement"].(float64)
	linkQuality, _ := deviceData["LinkQuality"].(float64)

	eventType := "unknown"
	if movement == 1 {
		eventType = "movement"
	}

	query := `
		INSERT INTO events (hub_id, device_id, event_type, link_quality, raw_data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`

	var eventID string
	err := p.db.QueryRow(ctx, query,
		hubID,
		deviceID,
		eventType,
		int(linkQuality),
		fmt.Sprintf("%v", eventData),
	).Scan(&eventID)

	if err != nil {
		p.logger.Errorw("Failed to save event", "error", err)
		return "", fmt.Errorf("save event: %w", err)
	}

	updateDeviceQuery := `
		UPDATE devices SET
			last_seen = EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - created_at)),
			last_seen_timestamp = CURRENT_TIMESTAMP,
			link_quality = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE device_id = $2
	`
	_, err = p.db.Exec(ctx, updateDeviceQuery, int(linkQuality), deviceID)
	if err != nil {
		p.logger.Warnw("Failed to update device last_seen", "error", err, "device_id", deviceID)
	}

	return eventID, nil
}

func (p *PGDB) GetEventsByUserID(ctx context.Context, userID, hours string) ([]models.Event, error) {
	hoursInt := 24
	if h, err := fmt.Sscanf(hours, "%d", &hoursInt); err != nil || h != 1 {
		hoursInt = 24
	}

	query := `
		SELECT 
			e.id::text,
			e.hub_id,
			e.device_id,
			e.event_type,
			e.link_quality,
			e.raw_data,
			e.created_at
		FROM events e
		INNER JOIN devices d ON e.device_id = d.device_id
		WHERE d.user_id = $1 
			AND e.created_at >= NOW() - ($2 || ' hours')::INTERVAL
		ORDER BY e.created_at DESC
	`

	rows, err := p.db.Query(ctx, query, userID, hoursInt)
	if err != nil {
		p.logger.Errorw("Failed to get events by user_id", "error", err)
		return nil, fmt.Errorf("get events by user: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		err := rows.Scan(
			&event.ID,
			&event.HubID,
			&event.DeviceID,
			&event.EventType,
			&event.LinkQuality,
			&event.RawData,
			&event.CreatedAt,
		)
		if err != nil {
			p.logger.Errorw("Failed to scan event", "error", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

func (p *PGDB) GetEventsByDeviceID(ctx context.Context, deviceID, hours string) ([]models.Event, error) {
	hoursInt := 24
	if h, err := fmt.Sscanf(hours, "%d", &hoursInt); err != nil || h != 1 {
		hoursInt = 24
	}

	query := `
		SELECT 
			id::text,
			hub_id,
			device_id,
			event_type,
			link_quality,
			raw_data,
			created_at
		FROM events 
		WHERE device_id = $1 
			AND created_at >= NOW() - ($2 || ' hours')::INTERVAL
		ORDER BY created_at DESC
	`

	rows, err := p.db.Query(ctx, query, deviceID, hoursInt)
	if err != nil {
		p.logger.Errorw("Failed to get events by device_id", "error", err)
		return nil, fmt.Errorf("get events by device: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		err := rows.Scan(
			&event.ID,
			&event.HubID,
			&event.DeviceID,
			&event.EventType,
			&event.LinkQuality,
			&event.RawData,
			&event.CreatedAt,
		)
		if err != nil {
			p.logger.Errorw("Failed to scan event", "error", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// ========== HUB MANAGEMENT ==========

func (p *PGDB) GetHubDevices(ctx context.Context, hubID string) ([]string, error) {
	query := `SELECT device_id FROM devices WHERE hub_id = $1`
	rows, err := p.db.Query(ctx, query, hubID)
	if err != nil {
		p.logger.Errorw("Failed to get hub devices", "error", err)
		return nil, fmt.Errorf("get hub devices: %w", err)
	}
	defer rows.Close()

	var deviceIDs []string
	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			p.logger.Errorw("Failed to scan device_id", "error", err)
			continue
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	return deviceIDs, nil
}

func (p *PGDB) GetUserHubID(ctx context.Context, userID string) (string, error) {
	var hubID sql.NullString
	query := `SELECT hub_id FROM users WHERE id = $1`
	err := p.db.QueryRow(ctx, query, userID).Scan(&hubID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user not found")
		}
		p.logger.Errorw("Failed to get user hub_id", "error", err)
		return "", fmt.Errorf("get user hub: %w", err)
	}

	if !hubID.Valid || hubID.String == "" {
		return "", fmt.Errorf("user has no hub assigned")
	}

	return hubID.String, nil
}

// GetActiveHubs возвращает список активных хабов (с привязанными пользователями)
func (p *PGDB) GetActiveHubs(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT hub_id 
		FROM devices 
		WHERE user_id IS NOT NULL 
		AND hub_id IS NOT NULL 
		AND hub_id != ''
		GROUP BY hub_id
		HAVING COUNT(*) > 0
	`

	rows, err := p.db.Query(ctx, query)
	if err != nil {
		p.logger.Errorw("Failed to get active hubs", "error", err)
		return nil, fmt.Errorf("get active hubs: %w", err)
	}
	defer rows.Close()

	var hubs []string
	for rows.Next() {
		var hubID string
		if err := rows.Scan(&hubID); err != nil {
			p.logger.Warnw("Failed to scan hub_id", "error", err)
			continue
		}
		hubs = append(hubs, hubID)
	}

	return hubs, nil
}

// GetHubUserID возвращает user_id для хаба
func (p *PGDB) GetHubUserID(ctx context.Context, hubID string) (string, error) {
	query := `
		SELECT user_id::text 
		FROM devices 
		WHERE hub_id = $1 AND user_id IS NOT NULL 
		LIMIT 1
	`

	var userID string
	err := p.db.QueryRow(ctx, query, hubID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no user found for hub %s", hubID)
		}
		p.logger.Errorw("Failed to get hub user_id", "hub_id", hubID, "error", err)
		return "", fmt.Errorf("get hub user: %w", err)
	}

	return userID, nil
}

func (p *PGDB) AutoAssignNewDevices(ctx context.Context, hubID string, deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var userID string
	queryGetUser := `
		SELECT user_id::text 
		FROM devices 
		WHERE hub_id = $1 AND user_id IS NOT NULL 
		LIMIT 1
	`

	err = tx.QueryRow(ctx, queryGetUser, hubID).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return fmt.Errorf("get hub user: %w", err)
	}

	queryAssign := `
		UPDATE devices 
		SET user_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE hub_id = $2 
			AND device_id = ANY($3) 
			AND (user_id IS NULL OR user_id = '')
	`

	result, err := tx.Exec(ctx, queryAssign, userID, hubID, deviceIDs)
	if err != nil {
		return fmt.Errorf("assign devices: %w", err)
	}

	rowsAffected := result.RowsAffected()

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	if rowsAffected > 0 {
		p.logger.Infow("Auto-assigned devices to user",
			"hub_id", hubID,
			"user_id", userID,
			"devices_assigned", rowsAffected,
			"total_devices", len(deviceIDs),
		)
	}

	return nil
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
