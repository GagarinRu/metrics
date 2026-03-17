package agent

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAgent_SendMetric(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, http.MethodPost, r.Method)
        assert.Equal(t, "/update/gauge/test/42.5", r.URL.Path)
        assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    agent := NewAgent(Config{
        ServerAddr: server.URL,
    })
    err := agent.sendMetric("gauge", "test", 42.5)
    assert.NoError(t, err)
}

func TestAgent_SendAllMetrics(t *testing.T) {
    receivedMetrics := make(map[string]bool)
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        receivedMetrics[r.URL.Path] = true
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    agent := NewAgent(Config{
        ServerAddr: server.URL,
    })
    agent.metrics.UpdateRuntimeMetrics()
    err := agent.sendAllMetrics()
    assert.NoError(t, err)
    assert.True(t, len(receivedMetrics) > 0)
}
