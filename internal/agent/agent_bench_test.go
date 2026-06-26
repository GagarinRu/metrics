package agent

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/metrics"
)

func BenchmarkCollectAllMetrics(b *testing.B) {
	a := NewAgent(Config{ServerAddr: "http://localhost:8080"})
	for i := 0; i < 10; i++ {
		a.metrics.UpdateRuntimeMetrics()
		a.metrics.UpdateSystemMetrics()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.collectAllMetrics()
	}
}

func BenchmarkCalculateHash(b *testing.B) {
	data := []byte(`[{"id":"Alloc","type":"gauge","value":123.45}]`)
	key := "agent-secret-key"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateHash(data, key)
	}
}

func BenchmarkUpdateRuntimeMetrics(b *testing.B) {
	m := metrics.NewMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.UpdateRuntimeMetrics()
	}
}
