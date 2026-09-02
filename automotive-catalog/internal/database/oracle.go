package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/sijms/go-ora/v2"

	"github.com/embuscado/automotive-catalog/pkg/config"
)

type OracleDB struct {
	DB *sql.DB
}

func NewOracle(ctx context.Context, cfg config.OracleConfig) (*OracleDB, error) {
	db, err := sql.Open("oracle", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("oracle: open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxOpenConns / 2)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("oracle: ping: %w", err)
	}

	return &OracleDB{DB: db}, nil
}

func (db *OracleDB) Close() error {
	return db.DB.Close()
}

func (db *OracleDB) HealthCheck(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}
