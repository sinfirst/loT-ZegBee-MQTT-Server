package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/config"
	"github.com/sinfirst/loT-ZegBee-MQTT-Server/internal/models"
	"go.uber.org/zap"
)

type PGDB struct {
	logger zap.SugaredLogger
	db     *pgxpool.Pool
}

func NewPGDB(conf config.Config, logger zap.SugaredLogger) *PGDB {
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

func (p *PGDB) AddDevice(ctx context.Context, sub models.Device) (int, error) {
	var id int
	query := `
		INSERT INTO subs (name_service, cost_per_month, user_uuid, date_start, date_end)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	start, end, err := dateParse(sub.StartDate, sub.EndDate)
	if err != nil {
		p.logger.Errorw("Problem with parse date: ", err)
		return 0, err
	}

	var endDB interface{}
	if end.IsZero() {
		endDB = nil
	} else {
		endDB = end
	}

	err = p.db.QueryRow(ctx, query, sub.ServiceName, sub.Price, sub.UserUUID, start, endDB).Scan(&id)
	if err != nil {
		p.logger.Errorw("Problem with create in db: ", err)
		return 0, err
	}

	return id, nil
}

func (p *PGDB) ReadFromDB(ctx context.Context, id string) (models.SubJSON, error) {
	var sub models.SubJSON
	var start time.Time
	var end sql.NullTime

	query := `SELECT id, name_service, cost_per_month, user_uuid, date_start, date_end FROM subs WHERE id = $1`
	row := p.db.QueryRow(ctx, query, id)
	err := row.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserUUID, &start, &end)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.SubJSON{}, fmt.Errorf("not found")
		}
		p.logger.Errorw("Problem with read from db: ", err)
		return models.SubJSON{}, err
	}

	sub.StartDate = timeToMonthYearString(start)
	if end.Valid {
		sub.EndDate = timeToMonthYearString(end.Time)
	} else {
		sub.EndDate = ""
	}

	return sub, nil
}

func (p *PGDB) UpdateInDB(ctx context.Context, sub models.SubJSON) error {
	exist, err := p.checkSubExistByID(ctx, sub.ID)
	if err != nil {
		p.logger.Errorw("Problem with check sub exist in db: ", err)
		return err
	}

	if !exist {
		return fmt.Errorf("not found")
	}

	start, end, err := dateParse(sub.StartDate, sub.EndDate)
	if err != nil {
		p.logger.Errorw("Problem with parse date: ", err)
		return err
	}

	var endDB interface{}
	if end.IsZero() {
		endDB = nil
	} else {
		endDB = end
	}

	query := "UPDATE subs SET "
	args := []interface{}{}
	paramCount := 0

	if sub.ServiceName != "" {
		paramCount++
		query += fmt.Sprintf("name_service = $%d, ", paramCount)
		args = append(args, sub.ServiceName)
	}

	if sub.Price != 0 {
		paramCount++
		query += fmt.Sprintf("cost_per_month = $%d, ", paramCount)
		args = append(args, sub.Price)
	}

	if sub.UserUUID != "" {
		paramCount++
		query += fmt.Sprintf("user_uuid = $%d, ", paramCount)
		args = append(args, sub.UserUUID)
	}

	if sub.StartDate != "" {
		paramCount++
		query += fmt.Sprintf("date_start = $%d, ", paramCount)
		args = append(args, start)
	}

	if sub.EndDate != "" {
		paramCount++
		query += fmt.Sprintf("date_end = $%d, ", paramCount)
		args = append(args, endDB)
	}

	query = strings.TrimSuffix(query, ", ")
	paramCount++
	query += fmt.Sprintf(" WHERE id = $%d", paramCount)
	args = append(args, sub.ID)

	result, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		p.logger.Errorw("Problem with update in db: ", err)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("not found")
	}

	return nil
}

func (p *PGDB) DeleteFromDB(ctx context.Context, id string) error {
	exist, err := p.checkSubExistByID(ctx, id)
	if err != nil {
		p.logger.Errorw("Problem with check sub exist in db: ", err)
		return err
	}

	if !exist {
		return fmt.Errorf("not found")
	}

	query := `DELETE FROM subs
				WHERE id = $1`

	_, err = p.db.Exec(ctx, query, id)

	if err != nil {
		p.logger.Errorw("Problem with deleting from db: ", err)
		return err
	}
	return nil
}

func (p *PGDB) ListFromDB(ctx context.Context, user_id string) ([]models.SubJSON, error) {
	var subs []models.SubJSON

	query := `SELECT id, name_service, cost_per_month, user_uuid, date_start, date_end FROM subs WHERE user_uuid = $1`
	rows, err := p.db.Query(ctx, query, user_id)

	if err != nil {
		p.logger.Errorw("Problem with create list from db: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sub models.SubJSON
		var start time.Time
		var end sql.NullTime

		err := rows.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserUUID, &start, &end)
		if err != nil {
			p.logger.Errorw("Problem with create list from db: ", err)
			return nil, err
		}

		sub.StartDate = timeToMonthYearString(start)
		sub.StartDate = timeToMonthYearString(start)
		if end.Valid {
			sub.EndDate = timeToMonthYearString(end.Time)
		} else {
			sub.EndDate = ""
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func (p *PGDB) CostSumSubFromDB(ctx context.Context, req models.SubJSON) (int, error) {
	var totalCost int
	query := `
        SELECT COALESCE(SUM(cost_per_month), 0) as total_cost
        FROM subs
        WHERE date_start >= $1 
        AND (date_end IS NULL OR date_end <= $2)
    `
	startTime, endTime, err := dateParse(req.StartDate, req.EndDate)
	if err != nil {
		p.logger.Errorw("Problem with parse date: ", err)
		return 0, err
	}

	args := []interface{}{startTime, endTime}
	argCounter := 3

	if req.UserUUID != "" {
		query += fmt.Sprintf(" AND user_uuid = $%d", argCounter)
		args = append(args, req.UserUUID)
		argCounter++
	}

	if req.ServiceName != "" {
		query += fmt.Sprintf(" AND name_service = $%d", argCounter)
		args = append(args, req.ServiceName)
		argCounter++
	}

	err = p.db.QueryRow(ctx, query, args...).Scan(&totalCost)
	if err != nil {
		p.logger.Errorw("Problem with calc cost: ", err)
		return 0, fmt.Errorf("failed to calculate total cost: %w", err)
	}

	return totalCost, nil
}

func (p *PGDB) checkSubExistByID(ctx context.Context, id string) (bool, error) {
	var exists bool

	err := p.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM subs WHERE id = $1
		)
	`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking user existence: %w", err)
	}
	return exists, nil
}

func InitMigrations(conf config.Config, logger zap.SugaredLogger) error {
	db, err := sql.Open("pgx", conf.Database.DataBaseDSN+"/"+conf.Database.Name)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.Up(db, "internal/storage/migrations"); err != nil {
		return err
	}

	logger.Infow("Migrations applied successfully")
	return nil
}

func timeToMonthYearString(t time.Time) string {
	return t.Format("01-2006")
}

func dateParse(start, end string) (time.Time, time.Time, error) {
	var startTime, endTime time.Time
	startTime, err := time.Parse("01-2006", start)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_month format: %w", err)
	}

	if end != "" {
		endTime, err = time.Parse("01-2006", end)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_month format: %w", err)
		}
		return startTime, endTime, nil
	}
	return startTime, time.Time{}, nil
}
