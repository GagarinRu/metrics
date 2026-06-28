package main

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/config"
	"github.com/stretchr/testify/require"
)

func TestServerConfigPriority(t *testing.T) {
	opts := config.ServerOptions{
		Address:         ":8080",
		StoreInterval:   300,
		FileStoragePath: "metrics.json",
	}
	file := config.ServerJSON{
		Address:     "localhost:9090",
		StoreFile:   "/from/json",
		CryptoKey:   "json-key",
	}
	merged, err := config.ApplyServerJSON(opts, file)
	require.NoError(t, err)
	require.Equal(t, "localhost:9090", merged.Address)

	merged.Address = "localhost:7070"
	merged.Key = "flag-key"
	t.Setenv("ADDRESS", "localhost:8081")
	t.Setenv("KEY", "env-key")
	merged = config.ApplyServerEnv(merged)
	require.Equal(t, "localhost:8081", merged.Address)
	require.Equal(t, "env-key", merged.Key)
}
