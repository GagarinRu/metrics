package metrics_test

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/metrics"
	"github.com/stretchr/testify/require"
)

func TestUpdateSystemMetrics(t *testing.T) {
	m := metrics.NewMetrics()
	m.UpdateSystemMetrics()
	m.UpdateRuntimeMetrics()

	require.NotZero(t, m.GetPollCount())
	require.NotZero(t, m.GetRandomValue())
}
