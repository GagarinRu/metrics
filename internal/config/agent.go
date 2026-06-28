package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// AgentJSON is the JSON config format for the metrics agent.
type AgentJSON struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
	LogLevel       string `json:"log_level"`
	RateLimit      int    `json:"rate_limit"`
}

// AgentOptions holds resolved agent configuration.
type AgentOptions struct {
	ServerAddr     string
	LogLevel       string
	PollInterval   int
	ReportInterval int
	Key            string
	CryptoKeyPath  string
	RateLimit      int
}

// ReadAgentJSON loads agent configuration from a JSON file.
func ReadAgentJSON(path string) (AgentJSON, error) {
	var file AgentJSON
	if err := loadJSON(path, &file); err != nil {
		return file, err
	}
	return file, nil
}

// ApplyAgentJSON merges JSON values into options.
func ApplyAgentJSON(opts AgentOptions, file AgentJSON) (AgentOptions, error) {
	if file.Address != "" {
		opts.ServerAddr = file.Address
	}
	if file.ReportInterval != "" {
		secs, err := parseDurationSeconds(file.ReportInterval)
		if err != nil {
			return opts, fmt.Errorf("report_interval: %w", err)
		}
		opts.ReportInterval = secs
	}
	if file.PollInterval != "" {
		secs, err := parseDurationSeconds(file.PollInterval)
		if err != nil {
			return opts, fmt.Errorf("poll_interval: %w", err)
		}
		opts.PollInterval = secs
	}
	if file.CryptoKey != "" {
		opts.CryptoKeyPath = file.CryptoKey
	}
	if file.LogLevel != "" {
		opts.LogLevel = file.LogLevel
	}
	if file.RateLimit > 0 {
		opts.RateLimit = file.RateLimit
	}
	return opts, nil
}

// ApplyAgentEnv applies environment variables to agent options.
func ApplyAgentEnv(opts AgentOptions) AgentOptions {
	opts.LogLevel = envString("LOG_LEVEL", opts.LogLevel)
	opts.ServerAddr = envString("ADDRESS", opts.ServerAddr)
	opts.PollInterval = envIntPositive("POLL_INTERVAL", opts.PollInterval)
	opts.ReportInterval = envIntPositive("REPORT_INTERVAL", opts.ReportInterval)
	opts.Key = envString("KEY", opts.Key)
	opts.CryptoKeyPath = envString("CRYPTO_KEY", opts.CryptoKeyPath)
	opts.RateLimit = envIntPositive("RATE_LIMIT", opts.RateLimit)
	return opts
}

func envIntPositive(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.Trim(val, `"'`)
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			return v
		}
		if d, err := time.ParseDuration(val); err == nil {
			secs := int(d.Seconds())
			if secs > 0 {
				return secs
			}
		}
	}
	return fallback
}
