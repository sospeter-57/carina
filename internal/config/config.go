package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	TLSEnabled      bool
	TLSCertFile     string
	TLSKeyFile      string
}

// PostgresConfig holds connection pool and database settings for Pgx / database/sql.
type PostgresConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// RedisConfig holds cache/queue connection settings.
type RedisConfig struct {
	Address  string
	Password string
	DBIndex  uint8
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret  string
	RefreshTTL time.Duration
}

// RateLimitsConfig holds rate limiting configuration.
type RateLimitsConfig struct {
	Count uint8
}

// StorageConfig holds MinIO / S3 Object Storage configuration.
type StorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

// Config aggregates all application configurations.
type Config struct {
	Server     ServerConfig
	Postgres   PostgresConfig
	Redis      RedisConfig
	Auth       AuthConfig
	RateLimits RateLimitsConfig
	Storage    StorageConfig
}

// Load reads environment variables into the Config struct.
func Load() (*Config, error) {
	// Load .env file if available (ignore error in production environments where env vars are set directly)
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, ErrMissingRequiredEnv("DATABASE_URL")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, ErrMissingRequiredEnv("JWT_SECRET")
	}

	cfg := &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnv("SERVER_PORT", "8080"),
			ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 20*time.Second),
			IdleTimeout:     getDuration("SERVER_IDLE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getDuration("SERVER_SHUTDOWN_TIMEOUT", 60*time.Second),
			TLSEnabled:      getBool("TLS_ENABLED", false),
			TLSCertFile:     getEnv("TLS_CERT_FILE", ""),
			TLSKeyFile:      getEnv("TLS_KEY_FILE", ""),
		},
		Postgres: PostgresConfig{
			URL:             dbURL,
			MaxConns:        int32(getInt("DB_MAX_CONNS", 25)),
			MinConns:        int32(getInt("DB_MIN_CONNS", 5)),
			MaxConnLifetime: getDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
			MaxConnIdleTime: getDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Address:  getEnv("REDIS_URL", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DBIndex:  uint8(getInt("REDIS_DB_INDEX", 0)),
		},
		Auth: AuthConfig{
			JWTSecret:  jwtSecret,
			RefreshTTL: getDuration("JWT_REFRESH_TTL", 20*time.Minute),
		},
		RateLimits: RateLimitsConfig{
			Count: uint8(getInt("RATE_LIMIT_COUNT", 30)),
		},
		Storage: StorageConfig{
			Endpoint:        getEnv("STORAGE_ENDPOINT", ""),
			Region:          getEnv("STORAGE_REGION", "us-east-1"),
			Bucket:          getEnv("STORAGE_BUCKET", ""),
			AccessKeyID:     getEnv("STORAGE_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("STORAGE_SECRET_ACCESS_KEY", ""),
			UseSSL:          getBool("STORAGE_USE_SSL", false),
		},
	}

	return cfg, nil
}

// --- Helper Functions ---

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

// Custom Error Types

type ErrMissingRequiredEnv string

func (e ErrMissingRequiredEnv) Error() string {
	return fmt.Sprintf("required environment variable not set: %s", string(e))
}