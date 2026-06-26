package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GagarinRu/metrics/internal/audit"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestHandlerNotifyAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	store := storage.NewMemStorage()
	auditor := audit.NewPublisher(path, "")
	defer func() { _ = auditor.Close() }()

	h := NewHandler(store, "", auditor)
	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/gauge/cpu/1", "text/plain", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(data), "cpu"))
}
