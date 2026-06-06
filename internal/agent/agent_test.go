package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GagarinRu/metrics/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestAgent_SendBatch_GaugeAndCounter(t *testing.T) {
	received := make(map[string]models.Metrics)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/updates", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var req []models.Metrics
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		for _, m := range req {
			received[m.ID] = m
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	agent := NewAgent(Config{
		ServerAddr: server.URL,
		UseGzip:    false,
	})
	gv := 42.5
	cv := int64(10)
	err := agent.sendBatch([]models.Metrics{
		{ID: "test", MType: "gauge", Value: &gv},
		{ID: "test_counter", MType: "counter", Delta: &cv},
	})
	assert.NoError(t, err)
	g, ok := received["test"]
	assert.True(t, ok)
	assert.Equal(t, "gauge", g.MType)
	assert.NotNil(t, g.Value)
	assert.Equal(t, 42.5, *g.Value)
	c, ok := received["test_counter"]
	assert.True(t, ok)
	assert.Equal(t, "counter", c.MType)
	assert.NotNil(t, c.Delta)
	assert.Equal(t, int64(10), *c.Delta)
}

func TestAgent_SendBatch(t *testing.T) {
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
	agent := NewAgent(Config{
		ServerAddr: server.URL,
		UseGzip:    false,
	})
	agent.metrics.UpdateRuntimeMetrics()
	agent.metrics.UpdateSystemMetrics()
	err := agent.sendBatch(agent.collectAllMetrics())
	assert.NoError(t, err)
	assert.True(t, len(receivedMetrics) > 0)
}

func TestAgent_Run(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgent(Config{
		ServerAddr:     server.URL,
		UseGzip:        false,
		PollInterval:   100 * time.Millisecond,
		ReportInterval: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := agent.Run(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
