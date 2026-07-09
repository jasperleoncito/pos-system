package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all environment-driven configuration for the application.
type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	MinIO    MinIOConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
	Xendit   XenditConfig
}

type AppConfig struct {
	Env  string // development | production | test
	Name string
}

type HTTPConfig struct {
	Port         string
	CORSOrigins  string // comma-separated browser origins allowed to call the API
	AppURL       string // single public base URL used in emailed links
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	// PublicBaseURL is the browser-facing base URL for stored objects,
	// e.g. http://localhost/storage
	PublicBaseURL string
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	FromName string
}

// XenditConfig holds the payment gateway credentials. WebhookToken is
// compared against the x-callback-token header on invoice callbacks.
type XenditConfig struct {
	SecretKey    string
	WebhookToken string
}

// Load reads configuration from environment variables and validates
// that required secrets are present.
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Name: getEnv("APP_NAME", "POS System"),
		},
		HTTP: HTTPConfig{
			Port:         getEnv("HTTP_PORT", "9137"),
			CORSOrigins:  getEnv("CORS_ORIGINS", "http://localhost:7642"),
			AppURL:       getEnv("APP_URL", ""),
			ReadTimeout:  getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "postgres"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "pos"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "pos"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: int32(getInt("DB_MAX_CONNS", 20)),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "redis:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		MinIO: MinIOConfig{
			Endpoint:      getEnv("MINIO_ENDPOINT", "minio:9000"),
			AccessKey:     getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey:     getEnv("MINIO_SECRET_KEY", ""),
			Bucket:        getEnv("MINIO_BUCKET", "pos"),
			UseSSL:        getBool("MINIO_USE_SSL", false),
			PublicBaseURL: getEnv("MINIO_PUBLIC_BASE_URL", "http://localhost/storage"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", ""),
			AccessTokenTTL:  getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTokenTTL: getDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "mailpit"),
			Port:     getEnv("SMTP_PORT", "1025"),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@pos.local"),
			FromName: getEnv("SMTP_FROM_NAME", "POS System"),
		},
		Xendit: XenditConfig{
			SecretKey:    getEnv("XENDIT_SECRET_KEY", ""),
			WebhookToken: getEnv("XENDIT_WEBHOOK_TOKEN", ""),
		},
	}

	// Emailed links need exactly one public URL; default to the first
	// allowed CORS origin when APP_URL isn't set explicitly.
	if cfg.HTTP.AppURL == "" {
		cfg.HTTP.AppURL = strings.TrimSpace(strings.Split(cfg.HTTP.CORSOrigins, ",")[0])
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.Database.Password == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if c.JWT.Secret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if c.MinIO.AccessKey == "" {
		missing = append(missing, "MINIO_ACCESS_KEY")
	}
	if c.MinIO.SecretKey == "" {
		missing = append(missing, "MINIO_SECRET_KEY")
	}
	// Billing can stay unconfigured in dev (checkout fails cleanly), but
	// production must never run without it.
	if c.App.IsProduction() {
		if c.Xendit.SecretKey == "" {
			missing = append(missing, "XENDIT_SECRET_KEY")
		}
		if c.Xendit.WebhookToken == "" {
			missing = append(missing, "XENDIT_WEBHOOK_TOKEN")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}
	return nil
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

func (c AppConfig) IsProduction() bool { return c.Env == "production" }

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
