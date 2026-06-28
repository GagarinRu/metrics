package crypto_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GagarinRu/metrics/internal/crypto"
	"github.com/stretchr/testify/require"
)

func TestLoadPublicKeyInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a pem"), 0o600))
	_, err := crypto.LoadPublicKey(path)
	require.Error(t, err)
}

func TestLoadPrivateKeyInvalidPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a pem"), 0o600))
	_, err := crypto.LoadPrivateKey(path)
	require.Error(t, err)
}

func TestDecryptNilKey(t *testing.T) {
	_, err := crypto.Decrypt([]byte{0, 1}, nil)
	require.Error(t, err)
}
