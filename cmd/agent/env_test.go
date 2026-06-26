package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetEnvString(t *testing.T) {
	t.Setenv("ADDRESS", "http://localhost:9090")
	require.Equal(t, "http://localhost:9090", getEnvString("ADDRESS", "http://localhost:8080"))
}

func TestGetEnvIntValid(t *testing.T) {
	t.Setenv("POLL_INTERVAL", "5")
	require.Equal(t, 5, getEnvInt("POLL_INTERVAL", 2))
}

func TestGetEnvIntDefault(t *testing.T) {
	require.Equal(t, 2, getEnvInt("UNKNOWN_POLL", 2))
}
