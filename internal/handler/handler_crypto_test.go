package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/metrics/internal/crypto"
	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestDecryptMiddleware(t *testing.T) {
	dir := t.TempDir()
	publicPath, privatePath := crypto.WriteTestKeyPair(t, dir)
	pub, err := crypto.LoadPublicKey(publicPath)
	require.NoError(t, err)
	priv, err := crypto.LoadPrivateKey(privatePath)
	require.NoError(t, err)

	store := storage.NewMemStorage()
	h := NewHandler(store, "", priv, nil)
	r := chi.NewRouter()
	r.Use(h.DecryptMiddleware)
	r.Post("/updates", h.UpdateMetricsBatch)

	body := []byte(`[{"id":"EncryptedMetric","type":"gauge","value":1.5}]`)
	encrypted, err := crypto.Encrypt(body, pub)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(encrypted))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	value, ok := store.GetGauge("EncryptedMetric")
	require.True(t, ok)
	require.Equal(t, 1.5, value)
}

func TestDecryptMiddlewarePlainBodyWithoutKey(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewHandler(store, "", nil, nil)
	r := chi.NewRouter()
	r.Use(h.DecryptMiddleware)
	r.Post("/updates", h.UpdateMetricsBatch)

	body := []byte(`[{"id":"PlainMetric","type":"gauge","value":2.5}]`)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDecryptMiddlewareInvalidCiphertext(t *testing.T) {
	dir := t.TempDir()
	_, privatePath := crypto.WriteTestKeyPair(t, dir)
	priv, err := crypto.LoadPrivateKey(privatePath)
	require.NoError(t, err)

	h := NewHandler(storage.NewMemStorage(), "", priv, nil)
	r := chi.NewRouter()
	r.Use(h.DecryptMiddleware)
	r.Post("/updates", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader([]byte("not-encrypted")))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEncryptedBatchEndToEnd(t *testing.T) {
	dir := t.TempDir()
	publicPath, privatePath := crypto.WriteTestKeyPair(t, dir)
	pub, err := crypto.LoadPublicKey(publicPath)
	require.NoError(t, err)
	priv, err := crypto.LoadPrivateKey(privatePath)
	require.NoError(t, err)

	store := storage.NewMemStorage()
	h := NewHandler(store, "", priv, nil)
	r := chi.NewRouter()
	r.Use(h.DecryptMiddleware)
	r.Post("/updates", h.UpdateMetricsBatch)
	srv := httptest.NewServer(r)
	defer srv.Close()

	gv := 99.9
	payload, err := json.Marshal([]models.Metrics{{ID: "E2E", MType: "gauge", Value: &gv}})
	require.NoError(t, err)
	encrypted, err := crypto.Encrypt(payload, pub)
	require.NoError(t, err)

	resp, err := http.Post(srv.URL+"/updates", "application/json", bytes.NewReader(encrypted))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, _ = io.Copy(io.Discard, resp.Body)

	value, ok := store.GetGauge("E2E")
	require.True(t, ok)
	require.Equal(t, 99.9, value)
}
