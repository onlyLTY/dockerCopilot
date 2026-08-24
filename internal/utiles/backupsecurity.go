package utiles

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const backupFormat = "docker-copilot-backup"

var backupAssociatedData = []byte("docker-copilot-backup-v1")

type encryptedBackupEnvelope struct {
	Format     string `json:"format"`
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func backupEncryptionSecret(accessSecret string) (string, error) {
	secret := os.Getenv("BACKUP_ENCRYPTION_KEY")
	if secret == "" {
		secret = accessSecret
	}
	return secret, nil
}

func encryptBackup(plaintext []byte, secret string) ([]byte, error) {
	key := sha256.Sum256([]byte("docker-copilot:" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, backupAssociatedData)
	return json.MarshalIndent(encryptedBackupEnvelope{
		Format: backupFormat, Version: 1, Algorithm: "AES-256-GCM",
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, "", "  ")
}

func decryptBackup(content []byte, secret string) ([]byte, error) {
	trimmed := strings.TrimSpace(string(content))
	if strings.HasPrefix(trimmed, "[") {
		// Backwards compatibility for backups created before encryption support.
		return content, nil
	}
	var envelope encryptedBackupEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return nil, errors.New("备份文件格式错误")
	}
	if envelope.Format != backupFormat || envelope.Version != 1 || envelope.Algorithm != "AES-256-GCM" {
		return nil, errors.New("不支持的备份文件格式")
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, errors.New("备份 nonce 格式错误")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, errors.New("备份密文格式错误")
	}
	key := sha256.Sum256([]byte("docker-copilot:" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("备份 nonce 长度错误")
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, backupAssociatedData)
	if err != nil {
		return nil, errors.New("备份解密失败，请检查 BACKUP_ENCRYPTION_KEY")
	}
	return plaintext, nil
}

func ensureBackupDir() (string, error) {
	dir, err := backupDirectoryPath()
	if err != nil {
		return "", err
	}
	// #nosec G301,G703 -- this is the explicitly configured backup root; 0700 is intentionally owner-only.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	// #nosec G302,G703 -- directories require execute permission; 0700 is the restrictive usable mode.
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func newBackupFilename(extension string) string {
	return fmt.Sprintf("backup-%s%s", time.Now().Format("2006-01-02T15-04-05.000000000"), extension)
}

func writeBackupAtomic(dir, filename string, content []byte) (retErr error) {
	if filename != filepath.Base(filename) || filename == "." {
		return errors.New("非法备份文件名")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	temporary, err := os.CreateTemp(dir, ".backup-*")
	if err != nil {
		return err
	}
	temporaryName := filepath.Base(temporary.Name())
	defer func() {
		_ = temporary.Close()
		if retErr != nil {
			_ = root.Remove(temporaryName)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return root.Rename(temporaryName, filename)
}
