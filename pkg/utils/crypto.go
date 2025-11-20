package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptSecret 使用 AES-256-GCM 对敏感信息加密
func EncryptSecret(key, plaintext string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("未配置 AES Key")
	}

	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("AES Key 长度必须为32字节")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret 使用 AES-256-GCM 解密敏感信息
func DecryptSecret(key, ciphertext string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("未配置 AES Key")
	}

	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("AES Key 长度必须为32字节")
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("密文长度非法")
	}

	nonce := raw[:gcm.NonceSize()]
	data := raw[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
