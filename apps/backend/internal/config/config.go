package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	DatabaseMaxConns  int32
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	AllowedOrigins    []string
	UploadDir         string
	MaxModelFileBytes int64
	MaxImageFileBytes int64
	StorageDriver     string
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKey       string
	S3SecretKey       string
	S3UseSSL          bool
}

func Load() (Config, error) {
	cfg := Config{
		Port:             env("PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DatabaseMaxConns: int32(numberEnv("DATABASE_MAX_CONNS", 10)),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AccessTokenTTL:   durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:  durationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		AllowedOrigins:   strings.Split(env("ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		UploadDir:        env("UPLOAD_DIR", "/data/uploads"),
		StorageDriver:    strings.ToLower(env("STORAGE_DRIVER", "local")),
		S3Endpoint:       strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		S3Region:         env("S3_REGION", "auto"),
		S3Bucket:         strings.TrimSpace(os.Getenv("S3_BUCKET")),
		S3AccessKey:      strings.TrimSpace(os.Getenv("S3_ACCESS_KEY_ID")),
		S3SecretKey:      strings.TrimSpace(os.Getenv("S3_SECRET_ACCESS_KEY")),
		S3UseSSL:         strings.EqualFold(env("S3_USE_SSL", "true"), "true"),
	}
	cfg.MaxModelFileBytes = mbEnv("MAX_MODEL_FILE_SIZE_MB", 200)
	cfg.MaxImageFileBytes = mbEnv("MAX_IMAGE_FILE_SIZE_MB", 10)
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 characters")
	}
	if cfg.StorageDriver != "local" && cfg.StorageDriver != "s3" {
		return Config{}, errors.New("STORAGE_DRIVER must be local or s3")
	}
	if cfg.StorageDriver == "s3" && (cfg.S3Endpoint == "" || cfg.S3Bucket == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "") {
		return Config{}, errors.New("S3 endpoint, bucket and credentials are required when STORAGE_DRIVER=s3")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func mbEnv(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(env(key, strconv.FormatInt(fallback, 10)), 10, 64)
	if err != nil || value < 1 {
		value = fallback
	}
	return value * 1024 * 1024
}

func numberEnv(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(env(key, strconv.FormatInt(fallback, 10)), 10, 32)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
