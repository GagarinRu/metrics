package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type MetricType string

const (
	MetricGauge   MetricType = "gauge"
	MetricCounter MetricType = "counter"
)

type Handler struct {
	storage storage.Storage
	key    string
}

func NewHandler(storage storage.Storage, key string) *Handler {
	return &Handler{storage: storage, key: key}
}

func (h *Handler) calculateHash(data []byte) []byte {
	hash := sha256.Sum256(append(data, []byte(h.key)...))
	return hash[:]
}

func (h *Handler) verifyHash(data []byte, hashHex string) bool {
	if h.key == "" {
		return true
	}
	expectedHash := h.calculateHash(data)
	receivedHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return false
	}
	if len(receivedHash) != len(expectedHash) {
		return false
	}
	for i := range expectedHash {
		if expectedHash[i] != receivedHash[i] {
			return false
		}
	}
	return true
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
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}
	if h.key != "" && r.Header.Get("HashSHA256") != "" {
		hashHeader := r.Header.Get("HashSHA256")
		if !h.verifyHash(bodyBytes, hashHeader) {
			http.Error(w, `{"error": "invalid hash"}`, http.StatusBadRequest)
			return
		}
	}
	var req models.Metrics
	dec := json.NewDecoder(bytes.NewReader(bodyBytes))
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
	if h.key != "" {
		hash := h.calculateHash(bodyBytes)
		w.Header().Set("HashSHA256", fmt.Sprintf("%x", hash))
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
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
	case MetricCounter:
		value, ok := h.storage.GetCounter(metricName)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatInt(value, 10)))
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}
}

func (h *Handler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}
	if h.key != "" && len(bodyBytes) > 0 {
		hashHeader := r.Header.Get("HashSHA256")
		if hashHeader != "" && !h.verifyHash(bodyBytes, hashHeader) {
			http.Error(w, `{"error": "invalid hash"}`, http.StatusBadRequest)
			return
		}
	}
	var req models.Metrics
	dec := json.NewDecoder(bytes.NewReader(bodyBytes))
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
	if h.key != "" {
		respBytes, _ := json.Marshal(resp)
		hash := h.calculateHash(respBytes)
		w.Header().Set("HashSHA256", fmt.Sprintf("%x", hash))
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

func (h *Handler) PingDataBase(w http.ResponseWriter, r *http.Request) {
	if err := h.storage.Ping(); err != nil {
		logger.Log.Error("Database ping failed", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) UpdateMetricsBatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}
	if h.key != "" {
		hashHeader := r.Header.Get("HashSHA256")
		if hashHeader == "" {
			http.Error(w, `{"error": "hash is required"}`, http.StatusBadRequest)
			return
		}
		if !h.verifyHash(bodyBytes, hashHeader) {
			http.Error(w, `{"error": "invalid hash"}`, http.StatusBadRequest)
			return
		}
	}
	var req []models.Metrics
	dec := json.NewDecoder(bytes.NewReader(bodyBytes))
	if err := dec.Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if len(req) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}
	for _, m := range req {
		if m.MType != string(MetricGauge) && m.MType != string(MetricCounter) {
			http.Error(w, `{"error": "invalid metric type"}`, http.StatusBadRequest)
			return
		}
		if m.ID == "" {
			http.Error(w, `{"error": "metric id is required"}`, http.StatusBadRequest)
			return
		}
		switch m.MType {
		case string(MetricGauge):
			if m.Value == nil {
				http.Error(w, `{"error": "value is required for gauge"}`, http.StatusBadRequest)
				return
			}
		case string(MetricCounter):
			if m.Delta == nil {
				http.Error(w, `{"error": "delta is required for counter"}`, http.StatusBadRequest)
				return
			}
		}
	}
	if err := h.storage.UpdateBatch(req); err != nil {
		logger.Log.Error("Failed to update batch", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}
	if h.key != "" {
		hash := h.calculateHash(bodyBytes)
		w.Header().Set("HashSHA256", fmt.Sprintf("%x", hash))
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
