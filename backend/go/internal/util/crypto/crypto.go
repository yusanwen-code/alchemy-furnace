// Package crypto 提供「炼丹炉」敏感信息的加解密工具
// 使用 AES-GCM 加密模型 API Key，密钥由 MODEL_KEY_SECRET 经 SHA256 归一化为 32 字节
// 密文格式: base64(nonce ‖ ciphertext)，nonce 为 12 字节随机数
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// ErrEmptySecret 加密密钥为空
var ErrEmptySecret = errors.New("加密密钥为空：请配置 MODEL_KEY_SECRET 环境变量")

// deriveKey 将任意长度的密钥字符串归一化为 32 字节 AES-256 密钥
func deriveKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// Encrypt 使用 AES-GCM 加密明文，返回 base64(nonce ‖ ciphertext)
// secret 为空时返回 ErrEmptySecret；明文为空时直接返回空字符串（无鉴权本地服务场景）
func Encrypt(plaintext, secret string) (string, error) {
	if secret == "" {
		return "", ErrEmptySecret
	}
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", fmt.Errorf("创建加密器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 字节随机 nonce
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机 nonce 失败: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密 Encrypt 生成的密文，返回明文
// secret 为空时返回 ErrEmptySecret；密文为空时直接返回空字符串
func Decrypt(encoded, secret string) (string, error) {
	if secret == "" {
		return "", ErrEmptySecret
	}
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("密文 base64 解码失败: %w", err)
	}

	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", fmt.Errorf("创建解密器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	if len(raw) < gcm.NonceSize() {
		return "", errors.New("密文长度不足，数据可能已损坏")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败（密钥不匹配或数据已损坏）: %w", err)
	}
	return string(plaintext), nil
}
