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

func (p *PGDB) AddDevices(ctx context.Context, devices []models.Device, hubID string) error {
	var id int
	query := `
		INSERT INTO devices (device_id, device_type, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	for _, device := range devices {

	}
	err := p.db.QueryRow(ctx, query, device.DeviceID, device.DeviceType, device.SensorStatus).Scan(&id)
	if err != nil {
		p.logger.Errorw("Problem with create in db: ", err)
		return err
	}

	return nil
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

func (p *PGDB) GetDevicesByUserID(ctx context.Context, userID string) ([]models.Device, error) {
	var devices []models.Device

	query := `SELECT device_id, hub_id, device_type, last_event, battery, signal_strength, orientation_state, sensor_status, last_seen FROM devices WHERE userID = $1`
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
