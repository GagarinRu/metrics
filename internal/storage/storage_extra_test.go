package storage_test

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_UpdateBatchMemory(t *testing.T) {
	store := storage.NewMemStorage()
	g1, g2 := 1.0, 2.0
	d1, d2 := int64(3), int64(4)

	err := store.UpdateBatch([]models.Metrics{
		{ID: "g1", MType: "gauge", Value: &g1},
		{ID: "g2", MType: "gauge", Value: &g2},
		{ID: "c1", MType: "counter", Delta: &d1},
		{ID: "c2", MType: "counter", Delta: &d2},
	})
	require.NoError(t, err)

	v, ok := store.GetGauge("g1")
	assert.True(t, ok)
	assert.Equal(t, 1.0, v)

	c, ok := store.GetCounter("c1")
	assert.True(t, ok)
	assert.Equal(t, int64(3), c)

	err = store.UpdateBatch([]models.Metrics{{ID: "c1", MType: "counter", Delta: &d2}})
	require.NoError(t, err)
	c, ok = store.GetCounter("c1")
	assert.True(t, ok)
	assert.Equal(t, int64(7), c)
}

func TestMemStorage_GetMissingMetric(t *testing.T) {
	store := storage.NewMemStorage()
	_, ok := store.GetGauge("missing")
	assert.False(t, ok)
	_, ok = store.GetCounter("missing")
	assert.False(t, ok)
}
