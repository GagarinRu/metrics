package handler

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRouter(key string) (*storage.MemStorage, *httptest.Server) {
	store := storage.NewMemStorage()
	h := NewHandler(store, key, nil)
	r := chi.NewRouter()
	r.Get("/", h.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
	r.Post("/update", h.UpdateMetricsJSON)
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Post("/value", h.GetMetricJSON)
	r.Get("/ping", h.PingDataBase)
	return store, httptest.NewServer(r)
}

func TestHandler_UpdateAndGetMetric(t *testing.T) {
	store, srv := newTestRouter("")
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/gauge/Alloc/42.5", "text/plain", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	val, ok := store.GetGauge("Alloc")
	assert.True(t, ok)
	assert.Equal(t, 42.5, val)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/value/gauge/Alloc", nil)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()
	body := make([]byte, 64)
	n, _ := resp.Body.Read(body)
	assert.Equal(t, "42.5", string(body[:n]))
}

func TestHandler_UpdateMetricsJSON(t *testing.T) {
	_, srv := newTestRouter("")
	defer srv.Close()

	payload := `{"id":"RandomValue","type":"gauge","value":0.75}`
	resp, err := http.Post(srv.URL+"/update", "application/json", strings.NewReader(payload))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandler_UpdateMetricsBatch(t *testing.T) {
	store, srv := newTestRouter("")
	defer srv.Close()

	gv := 1.1
	dl := int64(2)
	body, _ := json.Marshal([]models.Metrics{
		{ID: "g", MType: "gauge", Value: &gv},
		{ID: "c", MType: "counter", Delta: &dl},
	})
	resp, err := http.Post(srv.URL+"/updates", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	_, ok := store.GetGauge("g")
	assert.True(t, ok)
	_, ok = store.GetCounter("c")
	assert.True(t, ok)
}

func TestHandler_GetAllMetrics(t *testing.T) {
	store, srv := newTestRouter("")
	defer srv.Close()
	store.UpdateGauge("Alloc", 100)

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	assert.Contains(t, buf.String(), "Alloc")
}

func TestHandler_GetMetricJSON(t *testing.T) {
	store, srv := newTestRouter("")
	defer srv.Close()
	store.UpdateGauge("HeapAlloc", 512)

	payload := `{"id":"HeapAlloc","type":"gauge"}`
	resp, err := http.Post(srv.URL+"/value", "application/json", strings.NewReader(payload))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()

	var m models.Metrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&m))
	assert.Equal(t, "HeapAlloc", m.ID)
	require.NotNil(t, m.Value)
	assert.Equal(t, 512.0, *m.Value)
}

func TestHandler_InvalidMetricType(t *testing.T) {
	_, srv := newTestRouter("")
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/unknown/test/1", "text/plain", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandler_PingWithoutDB(t *testing.T) {
	_, srv := newTestRouter("")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ping")
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandler_HashVerification(t *testing.T) {
	_, srv := newTestRouter("secret")
	defer srv.Close()

	payload := []byte(`{"id":"test","type":"gauge","value":1}`)
	h := NewHandler(storage.NewMemStorage(), "secret", nil)
	hash := h.calculateHash(payload)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/update", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", hex.EncodeToString(hash))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}
