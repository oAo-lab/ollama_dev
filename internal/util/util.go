package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
)

// AES-GCM Key must be 16, 24, or 32 bytes long (AES-128, AES-192, AES-256)
const (
	keySize   = 32 // AES-256
	nonceSize = 12 // Recommended nonce size for AES-GCM
)

// Generate a secure random key
func generateKey() ([]byte, error) {
	key := make([]byte, keySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt a message using AES-GCM
func encrypt(key []byte, plaintext string) (string, error) {
	// Create a new AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Generate a random nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Create an AES-GCM instance
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Encrypt the plaintext
	ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Combine nonce and ciphertext into a single base64-encoded string
	combined := append(nonce, ciphertext...)
	return base64.URLEncoding.EncodeToString(combined), nil
}

// Decrypt a message using AES-GCM
func decrypt(key []byte, ciphertext string) (string, error) {
	// Decode the base64-encoded ciphertext
	decoded, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// Extract the nonce and ciphertext
	if len(decoded) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce := decoded[:nonceSize]
	encryptedText := decoded[nonceSize:]

	// Create an AES-GCM instance
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Decrypt the ciphertext
	plaintext, err := aesgcm.Open(nil, nonce, encryptedText, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func NewDecryptKey() (key []byte) {
	key, err := generateKey()
	if err != nil {
		slog.Warn("[generateKey] \t", "err", err)
		return
	}
	return key
}

func Encrypt(key []byte, data string) (string, error) {
	return encrypt(key, data)
}

func Decrypt(key []byte, data string) (string, error) {
	return decrypt(key, data)
}
