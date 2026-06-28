package logger_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	require.NoError(t, logger.Initialize("info"))
	require.NotNil(t, logger.Log)
}

func TestInitializeInvalidLevel(t *testing.T) {
	require.Error(t, logger.Initialize("not-a-level"))
}

func TestRequestLoggerWithBody(t *testing.T) {
	require.NoError(t, logger.Initialize("info"))
	handler := logger.RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
}
