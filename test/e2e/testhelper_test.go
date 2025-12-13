package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler"
	pgRepo "github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository/postgres"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/auth"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/middleware"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/server"
	authUC "github.com/marcos-nsantos/field-notes-backend/internal/usecase/auth"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/note"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/sync"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/upload"
)

const (
	testDBUser     = "testuser"
	testDBPassword = "testpass"
	testDBName     = "testdb"
	testJWTSecret  = "test-secret-key-for-e2e-tests"
	apiBasePath    = "/api/v1"
)

type TestApp struct {
	Server     *httptest.Server
	Pool       *pgxpool.Pool
	Container  testcontainers.Container
	BaseURL    string
	httpClient *http.Client
}

func setupTestApp(t *testing.T) *TestApp {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// Start PostgreSQL container with PostGIS
	pgContainer, err := postgres.Run(ctx,
		"postgis/postgis:18-3.6-alpine",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Run migrations
	err = runMigrations(ctx, pool)
	require.NoError(t, err)

	// Initialize repositories
	userRepo := pgRepo.NewUserRepo(pool)
	noteRepo := pgRepo.NewNoteRepo(pool)
	photoRepo := pgRepo.NewPhotoRepo(pool)
	deviceRepo := pgRepo.NewDeviceRepo(pool)
	refreshTokenRepo := pgRepo.NewRefreshTokenRepo(pool)

	// Initialize infrastructure services
	jwtSvc := auth.NewJWTService(testJWTSecret, 15*time.Minute)
	passwordHasher := auth.NewPasswordHasher(4) // Lower cost for faster tests

	// Stub storage for e2e tests (avoids S3 dependency)
	stubStorage := &stubImageStorage{}
	stubProcessor := &stubImageProcessor{}

	// Initialize use cases
	authSvc := authUC.NewService(userRepo, deviceRepo, refreshTokenRepo, jwtSvc, passwordHasher, 24*time.Hour)
	noteSvc := note.NewService(noteRepo, photoRepo)
	syncSvc := sync.NewService(noteRepo, deviceRepo)
	uploadSvc := upload.NewService(photoRepo, noteRepo, stubStorage, stubProcessor)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc)
	noteHandler := handler.NewNoteHandler(noteSvc)
	syncHandler := handler.NewSyncHandler(syncSvc)
	uploadHandler := handler.NewUploadHandler(uploadSvc)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtSvc)

	// Create router
	logger, _ := zap.NewDevelopment()
	router := server.NewRouter(server.RouterConfig{
		AuthHandler:    authHandler,
		NoteHandler:    noteHandler,
		SyncHandler:    syncHandler,
		UploadHandler:  uploadHandler,
		AuthMiddleware: authMiddleware,
		Logger:         logger,
		Environment:    "test",
	})

	// Create test server
	ts := httptest.NewServer(router.Engine())

	return &TestApp{
		Server:    ts,
		Pool:      pool,
		Container: pgContainer,
		BaseURL:   ts.URL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (app *TestApp) cleanup(t *testing.T) {
	t.Helper()

	app.Server.Close()
	app.Pool.Close()

	ctx := context.Background()
	if err := app.Container.Terminate(ctx); err != nil {
		t.Logf("failed to terminate container: %v", err)
	}
}

func (app *TestApp) request(method, path string, body any, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	fullPath := apiBasePath + path
	req, err := http.NewRequest(method, app.BaseURL+fullPath, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return app.httpClient.Do(req)
}

func (app *TestApp) get(path string, headers map[string]string) (*http.Response, error) {
	return app.request(http.MethodGet, path, nil, headers)
}

func (app *TestApp) post(path string, body any, headers map[string]string) (*http.Response, error) {
	return app.request(http.MethodPost, path, body, headers)
}

func (app *TestApp) put(path string, body any, headers map[string]string) (*http.Response, error) {
	return app.request(http.MethodPut, path, body, headers)
}

func (app *TestApp) delete(path string, headers map[string]string) (*http.Response, error) {
	return app.request(http.MethodDelete, path, nil, headers)
}

func parseResponse(t *testing.T, resp *http.Response, dest any) {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	if dest != nil {
		err = json.Unmarshal(body, dest)
		require.NoError(t, err, "response body: %s", string(body))
	}
}

func authHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

// Stub implementations for storage (to avoid S3 dependency in e2e tests)

type stubImageStorage struct{}

func (s *stubImageStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
	return nil
}

func (s *stubImageStorage) Delete(ctx context.Context, key string) error {
	return nil
}

func (s *stubImageStorage) GetURL(key string) string {
	return "https://stub-storage.example.com/" + key
}

func (s *stubImageStorage) GetSignedURL(key string, duration time.Duration) (string, error) {
	return "https://stub-storage.example.com/" + key + "?signed=true", nil
}

type stubImageProcessor struct{}

func (s *stubImageProcessor) Process(reader io.Reader, contentType string) (io.Reader, int64, int, int, error) {
	data, _ := io.ReadAll(reader)
	return bytes.NewReader(data), int64(len(data)), 800, 600, nil
}

// runMigrations executes database migrations for e2e tests
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
