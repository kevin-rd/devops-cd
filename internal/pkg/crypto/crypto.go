package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"

	"devops-cd/internal/pkg/config"
)

// HashPassword 哈希密码 (bcrypt)
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Encrypt AES加密
func Encrypt(plaintext string) (string, error) {
	key := []byte(config.GlobalConfig.Crypto.AESKey)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 转换为字节
	plaintextBytes := []byte(plaintext)

	// 创建GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	ciphertext := aesGCM.Seal(nonce, nonce, plaintextBytes, nil)

	// Base64编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt AES解密
func Decrypt(ciphertext string) (string, error) {
	key := []byte(config.GlobalConfig.Crypto.AESKey)

	// Base64解码
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 创建GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 获取nonce大小
	nonceSize := aesGCM.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", fmt.Errorf("密文太短")
	}

	// 提取nonce和密文
	nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]

	// 解密
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
