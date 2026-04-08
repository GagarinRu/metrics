package agent

import (
	"encoding/json"
	"github.com/GagarinRu/metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgent_SendMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/update", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var req models.Metrics
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, "test", req.ID)
		assert.Equal(t, "gauge", req.MType)
		assert.NotNil(t, req.Value)
		assert.Equal(t, 42.5, *req.Value)
		assert.Nil(t, req.Delta)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	agent := NewAgent(Config{
		ServerAddr: server.URL,
		UseGzip:    false,
	})
	err := agent.sendMetric("gauge", "test", 42.5)
	assert.NoError(t, err)
}

func TestAgent_SendMetricCounter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/update", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var req models.Metrics
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, "test_counter", req.ID)
		assert.Equal(t, "counter", req.MType)
		assert.NotNil(t, req.Delta)
		assert.Equal(t, int64(10), *req.Delta)
		assert.Nil(t, req.Value)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	agent := NewAgent(Config{
		ServerAddr: server.URL,
		UseGzip:    false,
	})
	err := agent.sendMetric("counter", "test_counter", int64(10))
	assert.NoError(t, err)
}

func TestAgent_SendAllMetricsJSON(t *testing.T) {
	receivedMetrics := make(map[string]models.Metrics)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/update", r.URL.Path)
		var req models.Metrics
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		receivedMetrics[req.ID] = req
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	useBatch := false
	agent := NewAgent(Config{
		ServerAddr: server.URL,
		UseGzip:    false,
		UseBatch:   &useBatch,
	})
	agent.metrics.UpdateRuntimeMetrics()
	err := agent.sendAllMetrics()
	assert.NoError(t, err)
	assert.True(t, len(receivedMetrics) > 0)
}

func TestAgent_SendAllMetricsBatch(t *testing.T) {
	receivedMetrics := make(map[string]models.Metrics)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/updates", r.URL.Path)
		var req []models.Metrics
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		for _, m := range req {
			receivedMetrics[m.ID] = m
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	useBatch := true
	agent := NewAgent(Config{
		ServerAddr: server.URL,
		UseGzip:    false,
		UseBatch:   &useBatch,
	})
	agent.metrics.UpdateRuntimeMetrics()
	err := agent.sendAllMetrics()
	assert.NoError(t, err)
	assert.True(t, len(receivedMetrics) > 0)
}
