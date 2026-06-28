package pool_test

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/pool"
	"github.com/stretchr/testify/require"
)

func TestPoolWithMetrics(t *testing.T) {
	p := pool.New[*models.Metrics]()

	m := &models.Metrics{ID: "test", MType: "gauge"}
	value := 1.5
	m.Value = &value

	p.Put(m)

	got := p.Get()
	require.NotNil(t, got)
	require.Empty(t, got.ID)
	require.Empty(t, got.MType)
}
