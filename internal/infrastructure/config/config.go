package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	S3        S3Config
	Log       LogConfig
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Port            int           `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout     time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"10s"`
	WriteTimeout    time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
	ShutdownTimeout time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"10s"`
	Environment     string        `envconfig:"ENVIRONMENT" default:"development"`
}

type DatabaseConfig struct {
	Host            string        `envconfig:"DB_HOST" default:"localhost"`
	Port            int           `envconfig:"DB_PORT" default:"5432"`
	User            string        `envconfig:"DB_USER" required:"true"`
	Password        string        `envconfig:"DB_PASSWORD" required:"true"`
	Name            string        `envconfig:"DB_NAME" required:"true"`
	SSLMode         string        `envconfig:"DB_SSL_MODE" default:"disable"`
	MaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

type JWTConfig struct {
	SecretKey       string        `envconfig:"JWT_SECRET_KEY" required:"true"`
	AccessTokenTTL  time.Duration `envconfig:"JWT_ACCESS_TOKEN_TTL" default:"15m"`
	RefreshTokenTTL time.Duration `envconfig:"JWT_REFRESH_TOKEN_TTL" default:"720h"`
}

type S3Config struct {
	Endpoint        string `envconfig:"S3_ENDPOINT"`
	Region          string `envconfig:"S3_REGION" default:"us-east-1"`
	Bucket          string `envconfig:"S3_BUCKET" required:"true"`
	AccessKeyID     string `envconfig:"S3_ACCESS_KEY_ID" required:"true"`
	SecretAccessKey string `envconfig:"S3_SECRET_ACCESS_KEY" required:"true"`
	UsePathStyle    bool   `envconfig:"S3_USE_PATH_STYLE" default:"false"`
	PublicURL       string `envconfig:"S3_PUBLIC_URL"`
}

type LogConfig struct {
	Level  string `envconfig:"LOG_LEVEL" default:"info"`
	Format string `envconfig:"LOG_FORMAT" default:"json"`
}

type RedisConfig struct {
	Host     string `envconfig:"REDIS_HOST" default:"localhost"`
	Port     int    `envconfig:"REDIS_PORT" default:"6379"`
	Password string `envconfig:"REDIS_PASSWORD" default:""`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type RateLimitConfig struct {
	Enabled         bool          `envconfig:"RATE_LIMIT_ENABLED" default:"true"`
	RequestsPerMin  int           `envconfig:"RATE_LIMIT_REQUESTS_PER_MIN" default:"100"`
	BurstSize       int           `envconfig:"RATE_LIMIT_BURST_SIZE" default:"10"`
	CleanupInterval time.Duration `envconfig:"RATE_LIMIT_CLEANUP_INTERVAL" default:"1m"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &cfg, nil
}
