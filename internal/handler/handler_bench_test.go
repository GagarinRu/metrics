package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func setupBenchHandler(metricCount int) (*Handler, *httptest.Server) {
	store := storage.NewMemStorage()
	for i := 0; i < metricCount; i++ {
		store.UpdateGauge("g"+string(rune('a'+i%26)), float64(i))
	}
	h := NewHandler(store, "bench-key", nil)
	r := chi.NewRouter()
	r.Get("/", h.GetAllMetrics)
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Post("/update", h.UpdateMetricsJSON)
	return h, httptest.NewServer(r)
}

func BenchmarkUpdateMetricsJSON(b *testing.B) {
	h, srv := setupBenchHandler(10)
	defer srv.Close()
	body := []byte(`{"id":"bench","type":"gauge","value":42.5}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		w := httptest.NewRecorder()
		h.UpdateMetricsJSON(w, req)
	}
}

func BenchmarkUpdateMetricsBatch(b *testing.B) {
	h, srv := setupBenchHandler(10)
	defer srv.Close()
	batch := make([]models.Metrics, 30)
	for i := range batch {
		v := float64(i)
		batch[i] = models.Metrics{ID: "m" + string(rune('a'+i%26)), MType: "gauge", Value: &v}
	}
	body, _ := json.Marshal(batch)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		w := httptest.NewRecorder()
		h.UpdateMetricsBatch(w, req)
	}
}

func BenchmarkGetAllMetrics(b *testing.B) {
	h, srv := setupBenchHandler(100)
	defer srv.Close()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.GetAllMetrics(w, req)
	}
}

func BenchmarkCalculateHash(b *testing.B) {
	h := NewHandler(storage.NewMemStorage(), "secret-key", nil)
	data := []byte(`{"id":"Alloc","type":"gauge","value":123.45}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.calculateHash(data)
	}
}
