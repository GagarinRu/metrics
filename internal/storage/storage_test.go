package storage

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GagarinRu/metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_UpdateAndGet(t *testing.T) {
	store := NewMemStorage()
	store.UpdateGauge("Alloc", 123.45)
	store.UpdateCounter("PollCount", 5)

	val, ok := store.GetGauge("Alloc")
	assert.True(t, ok)
	assert.Equal(t, 123.45, val)

	cnt, ok := store.GetCounter("PollCount")
	assert.True(t, ok)
	assert.Equal(t, int64(5), cnt)

	store.UpdateCounter("PollCount", 3)
	cnt, ok = store.GetCounter("PollCount")
	assert.True(t, ok)
	assert.Equal(t, int64(8), cnt)
}

func TestMemStorage_UpdateBatch(t *testing.T) {
	store := NewMemStorage()
	gv := 10.5
	dl := int64(7)
	err := store.UpdateBatch([]models.Metrics{
		{ID: "g1", MType: "gauge", Value: &gv},
		{ID: "c1", MType: "counter", Delta: &dl},
	})
	require.NoError(t, err)

	v, ok := store.GetGauge("g1")
	assert.True(t, ok)
	assert.Equal(t, 10.5, v)

	c, ok := store.GetCounter("c1")
	assert.True(t, ok)
	assert.Equal(t, int64(7), c)
}

func TestMemStorage_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")
	store := NewMemStorageWithFile(path, 0, false, "")
	store.UpdateGauge("HeapAlloc", 999.0)
	store.UpdateCounter("PollCount", 42)
	require.NoError(t, store.Save())

	restored := NewMemStorageWithFile(path, 0, true, "")
	v, ok := restored.GetGauge("HeapAlloc")
	assert.True(t, ok)
	assert.Equal(t, 999.0, v)
	c, ok := restored.GetCounter("PollCount")
	assert.True(t, ok)
	assert.Equal(t, int64(42), c)
}

func TestMemStorage_WriteMetricsHTML(t *testing.T) {
	store := NewMemStorage()
	store.UpdateGauge("Alloc", 100)
	store.UpdateCounter("PollCount", 3)

	var buf strings.Builder
	store.WriteMetricsHTML(&buf)
	html := buf.String()
	assert.Contains(t, html, "Alloc")
	assert.Contains(t, html, "100")
	assert.Contains(t, html, "PollCount")
	assert.Contains(t, html, "3")
}

func TestMemStorage_GetAll(t *testing.T) {
	store := NewMemStorage()
	store.UpdateGauge("g", 1.0)
	store.UpdateCounter("c", 2)

	gauges := store.GetAllGauges()
	counters := store.GetAllCounters()
	assert.Equal(t, 1.0, gauges["g"])
	assert.Equal(t, int64(2), counters["c"])
}

func TestMemStorage_PingWithoutDB(t *testing.T) {
	store := NewMemStorage()
	assert.Error(t, store.Ping())
}

func TestMemStorage_LoadMissingFile(t *testing.T) {
	store := NewMemStorageWithFile(filepath.Join(t.TempDir(), "missing.json"), 0, false, "")
	assert.NoError(t, store.Load())
}

func TestMemStorage_StopAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	store := NewMemStorageWithFile(path, 1, false, "")
	store.UpdateGauge("Alloc", 1)
	store.Stop()
	assert.NoError(t, store.Close())
}

func TestMemStorage_AutoSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	store := NewMemStorageWithFile(path, 1, false, "")
	store.UpdateCounter("PollCount", 1)
	time.Sleep(1100 * time.Millisecond)
	store.Stop()

	restored := NewMemStorageWithFile(path, 0, true, "")
	cnt, ok := restored.GetCounter("PollCount")
	assert.True(t, ok)
	assert.Equal(t, int64(1), cnt)
}
