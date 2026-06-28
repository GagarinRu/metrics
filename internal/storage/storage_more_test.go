package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetriableDBError(t *testing.T) {
	assert.False(t, isRetriableDBError(assert.AnError))
}

func TestMemStorage_Close(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	store := NewMemStorageWithFile(path, 0, false, "")
	store.UpdateGauge("x", 1)
	assert.NoError(t, store.Close())
}

func TestMemStorage_Stop(t *testing.T) {
	store := NewMemStorageWithFile(filepath.Join(t.TempDir(), "metrics.json"), 1, false, "")
	store.Stop()
}

func TestMemStorage_UpdateGaugeCounterMemoryOnly(t *testing.T) {
	store := NewMemStorage()
	store.UpdateGauge("g", 10)
	store.UpdateCounter("c", 5)
	store.UpdateCounter("c", 2)

	g, ok := store.GetGauge("g")
	assert.True(t, ok)
	assert.Equal(t, 10.0, g)

	cnt, ok := store.GetCounter("c")
	assert.True(t, ok)
	assert.Equal(t, int64(7), cnt)
}

func TestMemStorage_LoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not-json"), 0o644))
	store := NewMemStorageWithFile(path, 0, false, "")
	assert.Error(t, store.Load())
}
