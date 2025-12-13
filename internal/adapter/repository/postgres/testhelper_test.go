package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
	Pool      *pgxpool.Pool
	Container testcontainers.Container
}

func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgis/postgis:18-3.6-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	connString := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return &TestDB{
		Pool:      pool,
		Container: container,
	}
}

func (db *TestDB) Cleanup(t *testing.T) {
	t.Helper()
	if db.Pool != nil {
		db.Pool.Close()
	}
	if db.Container != nil {
		if err := db.Container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}
}

func (db *TestDB) Truncate(t *testing.T, tables ...string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range tables {
		_, err := db.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("failed to truncate table %s: %v", table, err)
		}
	}
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS postgis`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS notes (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			location GEOGRAPHY(POINT, 4326),
			altitude DOUBLE PRECISION,
			accuracy DOUBLE PRECISION,
			client_id VARCHAR(255),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ,
			UNIQUE(user_id, client_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_location ON notes USING GIST(location)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_updated_at ON notes(updated_at)`,
		`CREATE TABLE IF NOT EXISTS photos (
			id UUID PRIMARY KEY,
			note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			url VARCHAR(500) NOT NULL,
			key VARCHAR(255) NOT NULL,
			mime_type VARCHAR(100) NOT NULL,
			size BIGINT NOT NULL,
			width INT,
			height INT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_photos_note_id ON photos(note_id)`,
		`CREATE TABLE IF NOT EXISTS devices (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			device_id VARCHAR(255) NOT NULL,
			platform VARCHAR(50) NOT NULL,
			name VARCHAR(255) NOT NULL,
			sync_cursor TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, device_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
			token VARCHAR(255) NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			revoked_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_device_id ON refresh_tokens(device_id)`,
	}

	for _, migration := range migrations {
		if _, err := pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}
