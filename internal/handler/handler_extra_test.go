package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestHandler_HashMiddlewareInvalidHash(t *testing.T) {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "secret", nil)
	r := chi.NewRouter()
	r.Use(h.HashMiddleware)
	r.Post("/update", h.UpdateMetricsJSON)

	srv := httptest.NewServer(r)
	defer srv.Close()

	payload := `{"id":"x","type":"gauge","value":1}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/update", strings.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", "invalid")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandler_HashMiddlewareSkipsOtherPaths(t *testing.T) {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "secret", nil)
	r := chi.NewRouter()
	r.Use(h.HashMiddleware)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/gauge/test/1", "text/plain", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandler_GetMetricJSONNotFound(t *testing.T) {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "", nil)
	r := chi.NewRouter()
	r.Post("/value", h.GetMetricJSON)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/value", "application/json", bytes.NewReader([]byte(`{"id":"missing","type":"gauge"}`)))
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	_ = resp.Body.Close()
}
