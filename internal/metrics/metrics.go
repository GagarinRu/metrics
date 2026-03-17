package metrics

import (
    "math/rand"
    "runtime"
    "sync"
)

type MetricType string

const (
    MetricGauge   MetricType = "gauge"
    MetricCounter MetricType = "counter"
)

type Metrics struct {
    mu          sync.RWMutex
    gauges      map[string]float64
    counters    map[string]int64
    pollCount   int64
    randomValue float64
}

func NewMetrics() *Metrics {
    return &Metrics{
        gauges:   make(map[string]float64),
        counters: make(map[string]int64),
    }
}

func (m *Metrics) UpdateRuntimeMetrics() {
    m.mu.Lock()
    defer m.mu.Unlock()
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    m.gauges["Alloc"] = float64(memStats.Alloc)
    m.gauges["BuckHashSys"] = float64(memStats.BuckHashSys)
    m.gauges["Frees"] = float64(memStats.Frees)
    m.gauges["GCCPUFraction"] = memStats.GCCPUFraction
    m.gauges["GCSys"] = float64(memStats.GCSys)
    m.gauges["HeapAlloc"] = float64(memStats.HeapAlloc)
    m.gauges["HeapIdle"] = float64(memStats.HeapIdle)
    m.gauges["HeapInuse"] = float64(memStats.HeapInuse)
    m.gauges["HeapObjects"] = float64(memStats.HeapObjects)
    m.gauges["HeapReleased"] = float64(memStats.HeapReleased)
    m.gauges["HeapSys"] = float64(memStats.HeapSys)
    m.gauges["LastGC"] = float64(memStats.LastGC)
    m.gauges["Lookups"] = float64(memStats.Lookups)
    m.gauges["MCacheInuse"] = float64(memStats.MCacheInuse)
    m.gauges["MCacheSys"] = float64(memStats.MCacheSys)
    m.gauges["MSpanInuse"] = float64(memStats.MSpanInuse)
    m.gauges["MSpanSys"] = float64(memStats.MSpanSys)
    m.gauges["Mallocs"] = float64(memStats.Mallocs)
    m.gauges["NextGC"] = float64(memStats.NextGC)
    m.gauges["NumForcedGC"] = float64(memStats.NumForcedGC)
    m.gauges["NumGC"] = float64(memStats.NumGC)
    m.gauges["OtherSys"] = float64(memStats.OtherSys)
    m.gauges["PauseTotalNs"] = float64(memStats.PauseTotalNs)
    m.gauges["StackInuse"] = float64(memStats.StackInuse)
    m.gauges["StackSys"] = float64(memStats.StackSys)
    m.gauges["Sys"] = float64(memStats.Sys)
    m.gauges["TotalAlloc"] = float64(memStats.TotalAlloc)
    m.pollCount++
    m.randomValue = rand.Float64()
    m.counters["PollCount"] = m.pollCount
    m.gauges["RandomValue"] = m.randomValue
	
}

func (m *Metrics) GetGauges() map[string]float64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    gaugesCopy := make(map[string]float64, len(m.gauges))
    for k, v := range m.gauges {
        gaugesCopy[k] = v
    }
    return gaugesCopy
}

func (m *Metrics) GetCounters() map[string]int64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    countersCopy := make(map[string]int64, len(m.counters))
    for k, v := range m.counters {
        countersCopy[k] = v
    }
    return countersCopy
}

func (m *Metrics) GetPollCount() int64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.pollCount
}

func (m *Metrics) GetRandomValue() float64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.randomValue
}
