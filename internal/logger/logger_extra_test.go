package logger_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestRequestLogger(t *testing.T) {
	require.NoError(t, logger.Initialize("info"))

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := logger.RequestLogger(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
}
