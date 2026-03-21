package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
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
	metricType := MetricType(chi.URLParam(r, "metricType"))
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")
	switch metricType {
	case MetricGauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid gauge value", http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(metricName, value)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	case MetricCounter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid counter value", http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(metricName, value)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}
}

func (h *Handler) UpdateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.MType != string(MetricGauge) && req.MType != string(MetricCounter) {
		http.Error(w, `{"error": "invalid metric type"}`, http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, `{"error": "metric id is required"}`, http.StatusBadRequest)
		return
	}
	switch req.MType {
	case string(MetricGauge):
		if req.Value == nil {
			http.Error(w, `{"error": "value is required for gauge"}`, http.StatusBadRequest)
			return
		}
		h.storage.UpdateGauge(req.ID, *req.Value)
	case string(MetricCounter):
		if req.Delta == nil {
			http.Error(w, `{"error": "delta is required for counter"}`, http.StatusBadRequest)
			return
		}
		h.storage.UpdateCounter(req.ID, *req.Delta)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := MetricType(chi.URLParam(r, "metricType"))
	metricName := chi.URLParam(r, "metricName")
	switch metricType {
	case MetricGauge:
		value, ok := h.storage.GetGauge(metricName)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
	case MetricCounter:
		value, ok := h.storage.GetCounter(metricName)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatInt(value, 10)))
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.MType != string(MetricGauge) && req.MType != string(MetricCounter) {
		http.Error(w, `{"error": "invalid metric type"}`, http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, `{"error": "metric id is required"}`, http.StatusBadRequest)
		return
	}
	resp := models.Metrics{
		ID:    req.ID,
		MType: req.MType,
	}
	switch req.MType {
	case string(MetricGauge):
		value, ok := h.storage.GetGauge(req.ID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "metric not found"})
			return
		}
		resp.Value = &value
	case string(MetricCounter):
		value, ok := h.storage.GetCounter(req.ID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "metric not found"})
			return
		}
		resp.Delta = &value
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetAllMetrics(w http.ResponseWriter, r *http.Request) {
	gauges := h.storage.GetAllGauges()
	counters := h.storage.GetAllCounters()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<html><body>")
	fmt.Fprintf(w, "<h1>Metrics</h1>")
	fmt.Fprintf(w, "<h2>Gauges:</h2><ul>")
	for name, value := range gauges {
		fmt.Fprintf(w, "<li>%s: %v</li>", name, value)
	}
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "<h2>Counters:</h2><ul>")
	for name, value := range counters {
		fmt.Fprintf(w, "<li>%s: %v</li>", name, value)
	}
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "</body></html>")
}
