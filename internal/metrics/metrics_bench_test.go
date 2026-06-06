package metrics

import "testing"

func BenchmarkUpdateRuntimeMetrics(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.UpdateRuntimeMetrics()
	}
}

func BenchmarkUpdateSystemMetrics(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.UpdateSystemMetrics()
	}
}

func BenchmarkGetGauges(b *testing.B) {
	m := NewMetrics()
	m.UpdateRuntimeMetrics()
	m.UpdateSystemMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.GetGauges()
	}
}
