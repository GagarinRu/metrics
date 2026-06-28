package crypto_test

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/crypto"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	publicPath, privatePath := crypto.WriteTestKeyPair(t, dir)

	pub, err := crypto.LoadPublicKey(publicPath)
	require.NoError(t, err)
	priv, err := crypto.LoadPrivateKey(privatePath)
	require.NoError(t, err)

	plaintext := []byte(`[{"id":"Alloc","type":"gauge","value":123.45}]`)
	ciphertext, err := crypto.Encrypt(plaintext, pub)
	require.NoError(t, err)
	require.NotEqual(t, plaintext, ciphertext)

	decrypted, err := crypto.Decrypt(ciphertext, priv)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestLoadPublicKeyFromCertificate(t *testing.T) {
	dir := t.TempDir()
	publicPath, _ := crypto.WriteTestKeyPair(t, dir)
	pub, err := crypto.LoadPublicKey(publicPath)
	require.NoError(t, err)
	require.NotNil(t, pub)
	require.Equal(t, 2048, pub.N.BitLen())
}

func TestLoadPrivateKey(t *testing.T) {
	dir := t.TempDir()
	_, privatePath := crypto.WriteTestKeyPair(t, dir)
	priv, err := crypto.LoadPrivateKey(privatePath)
	require.NoError(t, err)
	require.NotNil(t, priv)
}

func TestDecryptInvalidData(t *testing.T) {
	dir := t.TempDir()
	_, privatePath := crypto.WriteTestKeyPair(t, dir)
	priv, err := crypto.LoadPrivateKey(privatePath)
	require.NoError(t, err)

	_, err = crypto.Decrypt([]byte{0, 0}, priv)
	require.Error(t, err)
}

func TestEncryptNilKey(t *testing.T) {
	_, err := crypto.Encrypt([]byte("test"), nil)
	require.Error(t, err)
}

func TestLoadPublicKeyInvalidFile(t *testing.T) {
	_, err := crypto.LoadPublicKey("nonexistent.pem")
	require.Error(t, err)
}

func TestLoadPrivateKeyInvalidFile(t *testing.T) {
	_, err := crypto.LoadPrivateKey("nonexistent.pem")
	require.Error(t, err)
}

func TestEncryptLargePayload(t *testing.T) {
	dir := t.TempDir()
	publicPath, privatePath := crypto.WriteTestKeyPair(t, dir)
	pub, err := crypto.LoadPublicKey(publicPath)
	require.NoError(t, err)
	priv, err := crypto.LoadPrivateKey(privatePath)
	require.NoError(t, err)

	plaintext := make([]byte, 64*1024)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	ciphertext, err := crypto.Encrypt(plaintext, pub)
	require.NoError(t, err)
	decrypted, err := crypto.Decrypt(ciphertext, priv)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}
