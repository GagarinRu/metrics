package metrics

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMetrics_UpdateRuntimeMetrics(t *testing.T) {
    m := NewMetrics()
    assert.Empty(t, m.GetGauges())
    assert.Empty(t, m.GetCounters())
    m.UpdateRuntimeMetrics()
    gauges := m.GetGauges()
    counters := m.GetCounters()
    assert.NotEmpty(t, gauges)
    assert.NotEmpty(t, counters)
    assert.Contains(t, gauges, "Alloc")
    assert.Contains(t, gauges, "RandomValue")
    assert.Contains(t, counters, "PollCount")
    assert.GreaterOrEqual(t, counters["PollCount"], int64(1))
    assert.GreaterOrEqual(t, gauges["RandomValue"], 0.0)
    assert.LessOrEqual(t, gauges["RandomValue"], 1.0)
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
    m := NewMetrics()
    done := make(chan bool)
    go func() {
        for i := 0; i < 100; i++ {
            m.UpdateRuntimeMetrics()
        }
        done <- true
    }()
    go func() {
        for i := 0; i < 100; i++ {
            m.GetGauges()
            m.GetCounters()
        }
        done <- true
    }()
}