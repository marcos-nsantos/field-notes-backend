package postgres_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/database"
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

	migrationsPath := getMigrationsPath()
	if err := database.RunMigrations(ctx, pool, migrationsPath); err != nil {
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

// getMigrationsPath returns the absolute path to the migrations directory
func getMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	repoDir := filepath.Dir(filename)
	return filepath.Join(repoDir, "..", "..", "..", "..", "migrations")
}
