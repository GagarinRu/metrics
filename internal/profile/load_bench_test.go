package profile

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/metrics"
	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func prepareStorage(metricCount int) *storage.MemStorage {
	store := storage.NewMemStorage()
	for i := 0; i < metricCount; i++ {
		store.UpdateGauge("gauge_"+string(rune('A'+i%26))+string(rune('0'+i/26)), float64(i)*1.5)
		store.UpdateCounter("counter_"+string(rune('A'+i%26))+string(rune('0'+i/26)), int64(i))
	}
	return store
}

func prepareBatch(metricCount int) []models.Metrics {
	batch := make([]models.Metrics, 0, metricCount*2)
	for i := 0; i < metricCount; i++ {
		v := float64(i) * 2.5
		d := int64(i)
		batch = append(batch,
			models.Metrics{ID: "batch_gauge_" + string(rune('a'+i%26)), MType: "gauge", Value: &v},
			models.Metrics{ID: "batch_counter_" + string(rune('a'+i%26)), MType: "counter", Delta: &d},
		)
	}
	return batch
}

func newTestServer(store *storage.MemStorage, key string) *httptest.Server {
	h := handler.NewHandler(store, key, nil)
	r := chi.NewRouter()
	r.Get("/", h.GetAllMetrics)
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Post("/update", h.UpdateMetricsJSON)
	r.Post("/value", h.GetMetricJSON)
	return httptest.NewServer(r)
}

func BenchmarkSystemLoad(b *testing.B) {
	store := prepareStorage(50)
	batch := prepareBatch(30)
	srv := newTestServer(store, "benchmark-secret-key")
	defer srv.Close()

	batchBody, err := json.Marshal(batch)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/updates", bytes.NewReader(batchBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()

		req, _ = http.NewRequest(http.MethodGet, srv.URL+"/", nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()

		store.GetAllGauges()
		store.GetAllCounters()
	}
}

func BenchmarkAgentCollect(b *testing.B) {
	m := metrics.NewMetrics()
	for i := 0; i < b.N; i++ {
		m.UpdateRuntimeMetrics()
		m.UpdateSystemMetrics()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		all := m.CollectAll()
		_, _ = json.Marshal(all)
	}
}
