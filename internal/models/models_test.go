package models_test

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/models"
	"github.com/stretchr/testify/require"
)

func TestMetricsReset(t *testing.T) {
	delta := int64(10)
	value := 3.14
	m := models.Metrics{
		ID:    "id",
		MType: "counter",
		Delta: &delta,
		Value: &value,
	}

	m.Reset()

	require.Empty(t, m.ID)
	require.Empty(t, m.MType)
	require.NotNil(t, m.Delta)
	require.Equal(t, int64(0), *m.Delta)
	require.NotNil(t, m.Value)
	require.Equal(t, 0.0, *m.Value)
}

func TestMetricsResetNil(t *testing.T) {
	var m *models.Metrics
	require.NotPanics(t, func() {
		m.Reset()
	})
}
