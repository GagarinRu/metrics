package handler

import (
    "net/http"
    "strconv"
    "strings"
    
    "github.com/GagarinRu/metrics/internal/storage"
)

type MetricType string

const (
    MetricGauge   MetricType = "gauge"
    MetricCounter MetricType = "counter"
)

type Handler struct {
    storage storage.Storage
}

func NewHandler(storage storage.Storage) *Handler {
    return &Handler{storage: storage}
}

func (h *Handler) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
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
            h.storage.UpdateGauge(metricName, value)
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("OK"))
        case MetricCounter:
            value, err := strconv.ParseInt(metricValue, 10, 64)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                return
            }
            h.storage.UpdateCounter(metricName, value)
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("OK"))
        default:
            w.WriteHeader(http.StatusBadRequest)
            return
    }
}
