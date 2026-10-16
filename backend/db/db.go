package db

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

//go:embed schema.sql
var schemaSQL string

func Init() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	pool, err := connectWithRetry(cfg, 10, 2*time.Second)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}

	// Важно: закрываем пул при ошибке инициализации схемы
	var initErr error
	defer func() {
		if initErr != nil && pool != nil {
			pool.Close()
		}
	}()

	ctx := context.Background()

	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		initErr = fmt.Errorf("apply schema: %w", err)
		return initErr
	}

	if _, err := pool.Exec(ctx, "TRUNCATE TABLE news RESTART IDENTITY"); err != nil {
		initErr = fmt.Errorf("clear news table: %w", err)
		return initErr
	}

	Pool = pool
	log.Println("Postgres pool connected, schema applied, news table cleared")
	return nil
}

func connectWithRetry(cfg *pgxpool.Config, attempts int, delay time.Duration) (*pgxpool.Pool, error) {
	var lastErr error

	for i := 1; i <= attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				cancel()
				return pool, nil
			} else {
				lastErr = pingErr
				pool.Close()
			}
		} else {
			lastErr = err
		}
		cancel()

		log.Printf("db connect attempt %d/%d failed: %v", i, attempts, lastErr)
		time.Sleep(delay)
	}

	return nil, lastErr
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
