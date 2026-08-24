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

func TestBackupEncryptionUsesUserChosenShortSecret(t *testing.T) {
	t.Setenv("BACKUP_ENCRYPTION_KEY", "")
	secret, err := backupEncryptionSecret("123456")
	if err != nil {
		t.Fatalf("user-chosen secret was rejected: %v", err)
	}
	if secret != "123456" {
		t.Fatalf("secret was modified: %q", secret)
	}
	plaintext := []byte(`[{"Name":"short-secret"}]`)
	encrypted, err := encryptBackup(plaintext, secret)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := decryptBackup(encrypted, secret)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round trip mismatch: %s", decrypted)
	}
}
