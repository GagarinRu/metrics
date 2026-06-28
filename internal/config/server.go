package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ServerJSON is the JSON config format for the metrics server.
type ServerJSON struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	LogLevel      string `json:"log_level"`
	AuditFile     string `json:"audit_file"`
	AuditURL      string `json:"audit_url"`
}

// ServerOptions holds resolved server configuration.
type ServerOptions struct {
	Address         string
	LogLevel        string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	CryptoKeyPath   string
	AuditFile       string
	AuditURL        string
}

// ReadServerJSON loads server configuration from a JSON file.
func ReadServerJSON(path string) (ServerJSON, error) {
	var file ServerJSON
	if err := loadJSON(path, &file); err != nil {
		return file, err
	}
	return file, nil
}

// ApplyServerJSON merges JSON values into options.
func ApplyServerJSON(opts ServerOptions, file ServerJSON) (ServerOptions, error) {
	if file.Address != "" {
		opts.Address = file.Address
	}
	opts.Restore = file.Restore
	if file.StoreInterval != "" {
		secs, err := parseDurationSeconds(file.StoreInterval)
		if err != nil {
			return opts, fmt.Errorf("store_interval: %w", err)
		}
		opts.StoreInterval = secs
	}
	if file.StoreFile != "" {
		opts.FileStoragePath = file.StoreFile
	}
	if file.DatabaseDSN != "" {
		opts.DatabaseDSN = file.DatabaseDSN
	}
	if file.CryptoKey != "" {
		opts.CryptoKeyPath = file.CryptoKey
	}
	if file.LogLevel != "" {
		opts.LogLevel = file.LogLevel
	}
	if file.AuditFile != "" {
		opts.AuditFile = file.AuditFile
	}
	if file.AuditURL != "" {
		opts.AuditURL = file.AuditURL
	}
	return opts, nil
}

// ApplyServerEnv applies environment variables to server options.
func ApplyServerEnv(opts ServerOptions) ServerOptions {
	opts.DatabaseDSN = envString("DATABASE_DSN", opts.DatabaseDSN)
	opts.Address = envString("ADDRESS", opts.Address)
	opts.LogLevel = envString("LOG_LEVEL", opts.LogLevel)
	opts.StoreInterval = envInt("STORE_INTERVAL", opts.StoreInterval)
	opts.FileStoragePath = envStringFirst("FILE_STORAGE_PATH", "STORE_FILE", opts.FileStoragePath)
	opts.Key = envString("KEY", opts.Key)
	opts.CryptoKeyPath = envString("CRYPTO_KEY", opts.CryptoKeyPath)
	opts.AuditFile = envString("AUDIT_FILE", opts.AuditFile)
	opts.AuditURL = envString("AUDIT_URL", opts.AuditURL)
	if envBool("RESTORE") {
		opts.Restore = true
	}
	return opts
}

func envString(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return strings.Trim(val, `"'`)
	}
	return fallback
}

func envStringFirst(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	fallback := keys[len(keys)-1]
	for _, key := range keys[:len(keys)-1] {
		if val, ok := os.LookupEnv(key); ok {
			return strings.Trim(val, `"'`)
		}
	}
	return fallback
}

func envBool(key string) bool {
	if val := os.Getenv(key); val != "" {
		if v, err := strconv.ParseBool(val); err == nil {
			return v
		}
	}
	return false
}

func envInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.Trim(val, `"'`)
		if v, err := strconv.Atoi(val); err == nil {
			return v
		}
		if d, err := time.ParseDuration(val); err == nil {
			return int(d.Seconds())
		}
	}
	return fallback
}
