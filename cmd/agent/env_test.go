package main

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/config"
	"github.com/stretchr/testify/require"
)

func TestAgentConfigPriority(t *testing.T) {
	opts := config.AgentOptions{
		ServerAddr:     "http://localhost:8080",
		PollInterval:   2,
		ReportInterval: 10,
	}
	file := config.AgentJSON{
		Address:        "http://localhost:9090",
		PollInterval:   "3s",
		ReportInterval: "5s",
	}
	merged, err := config.ApplyAgentJSON(opts, file)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:9090", merged.ServerAddr)
	require.Equal(t, 3, merged.PollInterval)

	t.Setenv("ADDRESS", "http://localhost:7070")
	t.Setenv("POLL_INTERVAL", "8")
	merged = config.ApplyAgentEnv(merged)
	require.Equal(t, "http://localhost:7070", merged.ServerAddr)
	require.Equal(t, 8, merged.PollInterval)
}
