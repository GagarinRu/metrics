package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GagarinRu/metrics/internal/config"
	"github.com/stretchr/testify/require"
)

func TestApplyServerJSON(t *testing.T) {
	opts := config.ServerOptions{
		Address:         ":8080",
		StoreInterval:   300,
		FileStoragePath: "metrics.json",
	}
	file := config.ServerJSON{
		Address:       "localhost:9090",
		Restore:       true,
		StoreInterval: "10s",
		StoreFile:     "/tmp/data.json",
		DatabaseDSN:   "postgres://localhost/db",
		CryptoKey:     "secret",
		LogLevel:      "debug",
	}
	merged, err := config.ApplyServerJSON(opts, file)
	require.NoError(t, err)
	require.Equal(t, "localhost:9090", merged.Address)
	require.True(t, merged.Restore)
	require.Equal(t, 10, merged.StoreInterval)
	require.Equal(t, "/tmp/data.json", merged.FileStoragePath)
	require.Equal(t, "postgres://localhost/db", merged.DatabaseDSN)
	require.Equal(t, "secret", merged.CryptoKeyPath)
	require.Equal(t, "debug", merged.LogLevel)
}

func TestApplyServerEnvOverridesJSON(t *testing.T) {
	opts := config.ServerOptions{
		Address:       ":8080",
		CryptoKeyPath: "from-file.pem",
	}
	t.Setenv("ADDRESS", "localhost:3000")
	t.Setenv("CRYPTO_KEY", "/env/private.pem")
	merged := config.ApplyServerEnv(opts)
	require.Equal(t, "localhost:3000", merged.Address)
	require.Equal(t, "/env/private.pem", merged.CryptoKeyPath)
}

func TestApplyServerEnvStoreFileAlias(t *testing.T) {
	opts := config.ServerOptions{FileStoragePath: "default.json"}
	t.Setenv("STORE_FILE", "/data/metrics.db")
	merged := config.ApplyServerEnv(opts)
	require.Equal(t, "/data/metrics.db", merged.FileStoragePath)
}

func TestReadServerJSONFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
		"address": "localhost:8080",
		"restore": true,
		"store_interval": "1s",
		"store_file": "/path/to/file.db"
	}`), 0o600))

	file, err := config.ReadServerJSON(path)
	require.NoError(t, err)
	require.Equal(t, "localhost:8080", file.Address)
	require.True(t, file.Restore)
	require.Equal(t, "1s", file.StoreInterval)
	require.Equal(t, "/path/to/file.db", file.StoreFile)
}

func TestApplyAgentJSON(t *testing.T) {
	opts := config.AgentOptions{
		ServerAddr:     "http://localhost:8080",
		PollInterval:   2,
		ReportInterval: 10,
	}
	file := config.AgentJSON{
		Address:        "http://localhost:9090",
		ReportInterval: "5s",
		PollInterval:   "3s",
		CryptoKey:      "key",
		LogLevel:       "debug",
		RateLimit:      3,
	}
	merged, err := config.ApplyAgentJSON(opts, file)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:9090", merged.ServerAddr)
	require.Equal(t, 3, merged.PollInterval)
	require.Equal(t, 5, merged.ReportInterval)
	require.Equal(t, "key", merged.CryptoKeyPath)
	require.Equal(t, "debug", merged.LogLevel)
	require.Equal(t, 3, merged.RateLimit)
}

func TestApplyAgentEnvOverridesJSON(t *testing.T) {
	opts := config.AgentOptions{
		PollInterval:    2,
		ReportInterval:  10,
		CryptoKeyPath:   "from-file.pem",
	}
	t.Setenv("POLL_INTERVAL", "7")
	t.Setenv("REPORT_INTERVAL", "15")
	t.Setenv("CRYPTO_KEY", "/env/public.pem")
	merged := config.ApplyAgentEnv(opts)
	require.Equal(t, 7, merged.PollInterval)
	require.Equal(t, 15, merged.ReportInterval)
	require.Equal(t, "/env/public.pem", merged.CryptoKeyPath)
}

func TestConfigPathFromEnv(t *testing.T) {
	t.Setenv("CONFIG", "/etc/metrics.json")
	require.Equal(t, "/etc/metrics.json", config.ConfigPath())
}

func TestParseDurationSeconds(t *testing.T) {
	opts := config.ServerOptions{StoreInterval: 300}
	file := config.ServerJSON{StoreInterval: "2s"}
	merged, err := config.ApplyServerJSON(opts, file)
	require.NoError(t, err)
	require.Equal(t, 2, merged.StoreInterval)
}
