package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetEnvString(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:9090")
	require.Equal(t, "localhost:9090", getEnvString("ADDRESS", ":8080"))
	require.Equal(t, ":8080", getEnvString("UNKNOWN", ":8080"))
}

func TestGetEnvBool(t *testing.T) {
	t.Setenv("RESTORE", "true")
	require.True(t, getEnvBool("RESTORE"))
	require.False(t, getEnvBool("UNKNOWN"))
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("STORE_INTERVAL", "120")
	require.Equal(t, 120, getEnvInt("STORE_INTERVAL", 300))

	t.Setenv("STORE_INTERVAL", "bad")
	require.Equal(t, 300, getEnvInt("STORE_INTERVAL", 300))
}

func TestGetEnvStringTrimsQuotes(t *testing.T) {
	t.Setenv("KEY", `"secret"`)
	require.Equal(t, "secret", getEnvString("KEY", ""))
}

func TestGetEnvStringFromOS(t *testing.T) {
	require.NoError(t, os.Setenv("FILE_STORAGE_PATH", "/tmp/metrics.json"))
	require.Equal(t, "/tmp/metrics.json", getEnvString("FILE_STORAGE_PATH", "metrics.json"))
}
