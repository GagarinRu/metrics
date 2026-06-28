package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const sessionKeyBytes = 32

var (
	// ErrTooShort is returned when ciphertext is shorter than the header.
	ErrTooShort = errors.New("ciphertext too short")
)

// Encrypt encrypts plaintext with hybrid RSA-OAEP + AES-GCM.
// Format: 2-byte RSA length || RSA ciphertext || AES-GCM ciphertext.
func Encrypt(plaintext []byte, pub *rsa.PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, errors.New("public key is nil")
	}
	sessionKey := make([]byte, sessionKeyBytes)
	if _, err := io.ReadFull(rand.Reader, sessionKey); err != nil {
		return nil, fmt.Errorf("generate session key: %w", err)
	}

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	rsaCiphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, sessionKey, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt session key: %w", err)
	}
	if len(rsaCiphertext) > 0xFFFF {
		return nil, errors.New("RSA ciphertext too long")
	}

	out := make([]byte, 2)
	binary.BigEndian.PutUint16(out, uint16(len(rsaCiphertext)))
	out = append(out, rsaCiphertext...)

	nonce := make([]byte, aead.NonceSize())
	out = aead.Seal(out, nonce, plaintext, nil)
	return out, nil
}

// Decrypt decrypts hybrid RSA-OAEP + AES-GCM ciphertext.
func Decrypt(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, errors.New("private key is nil")
	}
	if len(ciphertext) < 2 {
		return nil, ErrTooShort
	}
	rsaLen := int(binary.BigEndian.Uint16(ciphertext[:2]))
	if len(ciphertext) < rsaLen+2 {
		return nil, ErrTooShort
	}

	rsaCiphertext := ciphertext[2 : rsaLen+2]
	aesCiphertext := ciphertext[rsaLen+2:]

	sessionKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, rsaCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt session key: %w", err)
	}

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	plaintext, err := aead.Open(nil, nonce, aesCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}
	return plaintext, nil
}
