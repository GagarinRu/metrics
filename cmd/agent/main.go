package main

import (
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "sync"
)

type MetricType string

const (
    MetricGauge   MetricType = "gauge"
    MetricCounter MetricType = "counter"
)

type Storage interface {
    UpdateGauge(name string, value float64)
    UpdateCounter(name string, delta int64)
}

type MemStorage struct {
    mu      sync.RWMutex
    gauges  map[string]float64
    counters map[string]int64
}

func NewMemStorage() *MemStorage {
    return &MemStorage{
        gauges:   make(map[string]float64),
        counters: make(map[string]int64),
    }
}

func (s *MemStorage) UpdateGauge(name string, value float64) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.gauges[name] = value
}

func (s *MemStorage) UpdateCounter(name string, delta int64) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.counters[name] += delta
}

func updateHandler(storage Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            w.WriteHeader(http.StatusMethodNotAllowed)
			return
        }
        parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 4 || parts[0] != "update" {
			w.WriteHeader(http.StatusNotFound)
            return
        }
        metricType := MetricType(parts[1])
        metricName := parts[2]
        metricValue := parts[3]
        switch metricType {
			case MetricGauge:
				value, err := strconv.ParseFloat(metricValue, 64)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				storage.UpdateGauge(metricName, value)
			case MetricCounter:
				value, err := strconv.ParseInt(metricValue, 10, 64)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				storage.UpdateCounter(metricName, value)
			default:
				w.WriteHeader(http.StatusBadRequest)
				return
        }
        w.WriteHeader(http.StatusOK)
    }
}

func main() {
    storage := NewMemStorage()
    mux := http.NewServeMux()
    mux.HandleFunc("/update/", updateHandler(storage))
    fmt.Println("Server started on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        panic(err)
    }
}
