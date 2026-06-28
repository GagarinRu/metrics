package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestHandlerGetMetricCounterHTTP(t *testing.T) {
	store := storage.NewMemStorage()
	store.UpdateCounter("hits", 9)
	h := NewHandler(store, "", nil, nil)
	r := chi.NewRouter()
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/value/counter/hits")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandlerUpdateMetricsCounterHTTP(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store, "", nil, nil)
	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/counter/polls/3", "text/plain", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	cnt, ok := store.GetCounter("polls")
	require.True(t, ok)
	require.Equal(t, int64(3), cnt)
}

func TestHandlerUpdateMetricsJSONInvalidType(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store, "", nil, nil)
	r := chi.NewRouter()
	r.Post("/update", h.UpdateMetricsJSON)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update", "application/json", strings.NewReader(`{"id":"x","type":"invalid"}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandlerUpdateMetricsJSONMissingID(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store, "", nil, nil)
	r := chi.NewRouter()
	r.Post("/update", h.UpdateMetricsJSON)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update", "application/json", strings.NewReader(`{"id":"","type":"gauge","value":1}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandlerUpdateMetricsJSONMissingGaugeValue(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store, "", nil, nil)
	r := chi.NewRouter()
	r.Post("/update", h.UpdateMetricsJSON)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update", "application/json", strings.NewReader(`{"id":"g","type":"gauge"}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandlerUpdateMetricsJSONMissingCounterDelta(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store, "", nil, nil)
	r := chi.NewRouter()
	r.Post("/update", h.UpdateMetricsJSON)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update", "application/json", strings.NewReader(`{"id":"c","type":"counter"}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	_ = resp.Body.Close()
}
