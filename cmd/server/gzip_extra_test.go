package main

import (
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

func TestGzipMiddlewareInvalidBody(t *testing.T) {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "", nil)
	r := chi.NewRouter()
	r.Post("/update", h.UpdateMetricsJSON)

	req := httptest.NewRequest(http.MethodPost, "/update", io.NopCloser(strings.NewReader("not-gzip")))
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	gzipMiddleware(r).ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
