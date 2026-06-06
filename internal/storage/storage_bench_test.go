package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GagarinRu/metrics/internal/models"
)

func BenchmarkUpdateGauge(b *testing.B) {
	store := NewMemStorage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.UpdateGauge("Alloc", float64(i))
	}
}

func BenchmarkUpdateCounter(b *testing.B) {
	store := NewMemStorage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.UpdateCounter("PollCount", 1)
	}
}

func BenchmarkUpdateBatch(b *testing.B) {
	store := NewMemStorage()
	batch := make([]models.Metrics, 50)
	for i := range batch {
		v := float64(i)
		batch[i] = models.Metrics{ID: "metric", MType: "gauge", Value: &v}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.UpdateBatch(batch)
	}
}

func BenchmarkGetAllGauges(b *testing.B) {
	store := NewMemStorage()
	for i := 0; i < 100; i++ {
		store.UpdateGauge("gauge_"+string(rune('A'+i%26)), float64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.GetAllGauges()
	}
}

func BenchmarkSave(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "metrics.json")
	store := NewMemStorageWithFile(path, 0, false, "")
	for i := 0; i < 100; i++ {
		store.UpdateGauge("gauge_"+string(rune('A'+i%26)), float64(i))
		store.UpdateCounter("counter_"+string(rune('A'+i%26)), int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.Save(); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	_ = os.Remove(path)
}
