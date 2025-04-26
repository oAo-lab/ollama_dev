package util

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate a test key
	key := NewDecryptKey()

	// Test data
	plaintext := "Hello, secure world!"

	// Encrypt the plaintext
	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	t.Logf("Encrypted: %s", encrypted)

	// Decrypt the ciphertext
	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	t.Logf("Decrypted: %s", decrypted)

	// Verify that the decrypted text matches the original plaintext
	if decrypted != plaintext {
		t.Errorf("Decrypted text does not match the original plaintext.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	// Generate a test key
	key := NewDecryptKey()

	// Invalid ciphertext (shorter than nonce size)
	invalidCiphertext := "short"

	// Attempt to decrypt the invalid ciphertext
	msg, err := Decrypt(key, invalidCiphertext)
	if err != nil {
		t.Log("err: ", err)
	}
	t.Log("msg: ", msg)
}

func TestDecryptEmptyCiphertext(t *testing.T) {
	// Generate a test key
	key := NewDecryptKey()

	// Empty ciphertext
	emptyCiphertext := ""

	// Attempt to decrypt the empty ciphertext
	msg, err := Decrypt(key, emptyCiphertext)
	if err == nil {
		t.Error("Expected an error for empty ciphertext, but got none")
	}

	t.Log("msg: ", msg)
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	// Generate a test key
	key := NewDecryptKey()

	// Test data: empty string
	plaintext := ""

	// Encrypt the plaintext
	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	t.Logf("Encrypted (empty string): %s", encrypted)

	// Decrypt the ciphertext
	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	t.Logf("Decrypted (empty string): %s", decrypted)

	// Verify that the decrypted text matches the original plaintext
	if decrypted != plaintext {
		t.Errorf("Decrypted text does not match the original plaintext.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}

func TestEncryptDecryptLongString(t *testing.T) {
	// Generate a test key
	key := NewDecryptKey()

	// Test data: a long string
	plaintext := "This is a very long string that will be encrypted and decrypted using AES-GCM. " +
		"It should handle large inputs without any issues."

	// Encrypt the plaintext
	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	t.Logf("Encrypted (long string): %s", encrypted)

	// Decrypt the ciphertext
	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	t.Logf("Decrypted (long string): %s", decrypted)

	// Verify that the decrypted text matches the original plaintext
	if decrypted != plaintext {
		t.Errorf("Decrypted text does not match the original plaintext.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}

func TestEncryptDecryptWithDifferentKeys(t *testing.T) {
	// Generate two different keys
	key1 := NewDecryptKey()

	key2 := NewDecryptKey()

	// Test data
	plaintext := "Hello, secure world!"

	// Encrypt the plaintext with key1
	encrypted, err := Encrypt(key1, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	t.Logf("Encrypted: %s", encrypted)

	// Attempt to decrypt the ciphertext with key2 (should fail)
	_, err = Decrypt(key2, encrypted)
	if err == nil {
		t.Error("Expected an error when decrypting with a different key, but got none")
	}
}

func TestEncryptDecryptWithEmptyKey(t *testing.T) {
	// Empty key (invalid key size)
	emptyKey := []byte{}

	// Test data
	plaintext := "Hello, secure world!"

	// Attempt to encrypt with an empty key
	_, err := Encrypt(emptyKey, plaintext)
	if err == nil {
		t.Error("Expected an error when encrypting with an empty key, but got none")
	}

	// Attempt to decrypt with an empty key
	_, err = Decrypt(emptyKey, "dummy-ciphertext")
	if err == nil {
		t.Error("Expected an error when decrypting with an empty key, but got none")
	}
}

func TestEncryptDecryptWithInvalidKeySize(t *testing.T) {
	// Invalid key size (15 bytes, not 16, 24, or 32)
	invalidKey := make([]byte, 15)

	// Test data
	plaintext := "Hello, secure world!"

	// Attempt to encrypt with an invalid key size
	_, err := Encrypt(invalidKey, plaintext)
	if err == nil {
		t.Error("Expected an error when encrypting with an invalid key size, but got none")
	}

	// Attempt to decrypt with an invalid key size
	_, err = Decrypt(invalidKey, "dummy-ciphertext")
	if err == nil {
		t.Error("Expected an error when decrypting with an invalid key size, but got none")
	}
}
