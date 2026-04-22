package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestGzipCompression(t *testing.T) {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "")
	r := chi.NewRouter()
	r.Get("/", h.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
	r.Post("/update", h.UpdateMetricsJSON)
	r.Post("/value", h.GetMetricJSON)
	handlerWithGzip := gzipMiddleware(r)
	srv := httptest.NewServer(handlerWithGzip)
	defer srv.Close()
	t.Run("sends_gzip", func(t *testing.T) {
		reqBody := `{"id":"test","type":"counter","delta":5}`
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write([]byte(reqBody))
		require.NoError(t, err)
		err = zw.Close()
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, srv.URL+"/update", &buf)
		r.RequestURI = ""
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()
		require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		defer zr.Close()
		b, err := io.ReadAll(zr)
		require.NoError(t, err)
		var respBody map[string]interface{}
		err = json.Unmarshal(b, &respBody)
		require.NoError(t, err)
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		reqBody := `{"id":"test_get","type":"gauge","value":123.45}`
		r := httptest.NewRequest(http.MethodPost, srv.URL+"/update", strings.NewReader(reqBody))
		r.RequestURI = ""
		r.Header.Set("Content-Type", "application/json")
		_, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		r = httptest.NewRequest(http.MethodGet, srv.URL+"/value/gauge/test_get", nil)
		r.RequestURI = ""
		r.Header.Set("Accept-Encoding", "gzip")
		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()
		require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		defer zr.Close()
		b, err := io.ReadAll(zr)
		require.NoError(t, err)
		require.Equal(t, "123.45", string(b))
	})
	t.Run("no_gzip", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, srv.URL+"/update/counter/test2/10", nil)
		r.RequestURI = ""
		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()
		require.Empty(t, resp.Header.Get("Content-Encoding"))
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(b), "OK")
	})
}
