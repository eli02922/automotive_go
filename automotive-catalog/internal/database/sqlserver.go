package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/embuscado/automotive-catalog/pkg/config"
)

type SQLServerDB struct {
	DB *sql.DB
}

func NewSQLServer(ctx context.Context, cfg config.MSSQLConfig) (*SQLServerDB, error) {
	db, err := sql.Open("sqlserver", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sqlserver: open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxOpenConns / 2)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("sqlserver: ping: %w", err)
	}

	return &SQLServerDB{DB: db}, nil
}

func (db *SQLServerDB) Close() error {
	return db.DB.Close()
}

func (db *SQLServerDB) HealthCheck(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}
