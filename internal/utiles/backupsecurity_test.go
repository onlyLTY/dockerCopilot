package utiles

import (
	"bytes"
	"testing"
)

func TestBackupEncryptionRoundTrip(t *testing.T) {
	plaintext := []byte(`[{"Config":{"Env":["PASSWORD=secret"]}}]`)
	encrypted, err := encryptBackup(plaintext, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("PASSWORD")) || bytes.Contains(encrypted, []byte("secret")) {
		t.Fatalf("encrypted backup contains plaintext secret: %s", encrypted)
	}
	decrypted, err := decryptBackup(encrypted, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round trip mismatch: %s", decrypted)
	}
	if _, err := decryptBackup(encrypted, "wrong secret"); err == nil {
		t.Fatal("backup decrypted with wrong key")
	}
}

func TestDecryptBackupSupportsLegacyPlaintext(t *testing.T) {
	legacy := []byte("  [ {\"Name\":\"legacy\"} ]")
	decrypted, err := decryptBackup(legacy, "unused secret")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, legacy) {
		t.Fatalf("legacy backup changed: %s", decrypted)
	}
}
