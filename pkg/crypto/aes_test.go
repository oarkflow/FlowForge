package crypto

import (
	"crypto/rand"
	"strings"
	"testing"
)

func validKey() []byte {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := validKey()
	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"short string", "hello"},
		{"unicode", "Unicode: \u4e16\u754c"},
		{"long string", strings.Repeat("abcdefghij", 1000)},
		{"special chars", `{"key": "value", "arr": [1,2,3]}`},
		{"newlines", "line1\nline2\nline3"},
		{"binary-like", "\x00\x01\x02\xff\xfe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt(key, tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if ciphertext == tt.plaintext && tt.plaintext != "" {
				t.Fatal("Encrypt() returned plaintext unchanged")
			}

			decrypted, err := Decrypt(key, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("round-trip mismatch: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := validKey()
	plaintext := "same plaintext"

	ct1, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	ct2, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	if ct1 == ct2 {
		t.Error("Encrypt() produced identical ciphertexts for the same plaintext (nonce not random)")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := validKey()
	key2 := validKey()

	ciphertext, err := Encrypt(key1, "secret data")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(key2, ciphertext)
	if err == nil {
		t.Error("Decrypt() with wrong key should return an error")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	key := validKey()
	_, err := Decrypt(key, "not-valid-base64!!!")
	if err == nil {
		t.Error("Decrypt() with invalid base64 should return an error")
	}
}

func TestDecryptTooShortCiphertext(t *testing.T) {
	key := validKey()
	// Encrypt something, then truncate the base64 string
	_, err := Decrypt(key, "AAAA") // very short valid base64
	if err == nil {
		t.Error("Decrypt() with too-short ciphertext should return an error")
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	badKey := []byte("short")
	_, err := Encrypt(badKey, "plaintext")
	if err == nil {
		t.Error("Encrypt() with invalid key length should return an error")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := validKey()
	ciphertext, err := Encrypt(key, "original")
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the last character of the base64 string
	tampered := ciphertext[:len(ciphertext)-2] + "XX"
	_, err = Decrypt(key, tampered)
	if err == nil {
		t.Error("Decrypt() with tampered ciphertext should return an error")
	}
}
