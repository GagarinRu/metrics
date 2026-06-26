package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestHandler_UpdateMetricsBatchEmpty(t *testing.T) {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "", nil)
	r := chi.NewRouter()
	r.Post("/updates", h.UpdateMetricsBatch)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/updates", "application/json", bytes.NewReader([]byte("[]")))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestHandler_GetMetricNotFound(t *testing.T) {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "", nil)
	r := chi.NewRouter()
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/value/gauge/missing")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	_ = resp.Body.Close()
}
