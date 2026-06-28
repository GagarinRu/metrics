package handler

import (
	"encoding/hex"
	"testing"

	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestHandlerVerifyHash(t *testing.T) {
	h := NewHandler(storage.NewMemStorage(), "secret", nil)
	data := []byte(`{"id":"x","type":"gauge","value":1}`)
	hash := hex.EncodeToString(h.calculateHash(data))
	require.True(t, h.verifyHash(data, hash))
	require.False(t, h.verifyHash(data, "invalid"))
}
