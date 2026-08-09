// Package crypto AES-GCM 加解密工具单元测试
package crypto

import (
	"strings"
	"testing"
)

// TestEncryptDecryptRoundtrip 加密后可正确解密还原
func TestEncryptDecryptRoundtrip(t *testing.T) {
	secret := "test-secret-key"
	plaintext := "sk-abcdef1234567890"

	encrypted, err := Encrypt(plaintext, secret)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if encrypted == "" || encrypted == plaintext {
		t.Fatalf("密文不应为空或等于明文: %q", encrypted)
	}
	if strings.Contains(encrypted, plaintext) {
		t.Fatalf("密文不应包含明文: %q", encrypted)
	}

	decrypted, err := Decrypt(encrypted, secret)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("解密结果不符: got %q, want %q", decrypted, plaintext)
	}
}

// TestEncryptRandomNonce 同一明文两次加密结果应不同（随机 nonce）
func TestEncryptRandomNonce(t *testing.T) {
	a, err := Encrypt("same-plaintext", "secret")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	b, err := Encrypt("same-plaintext", "secret")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if a == b {
		t.Fatal("两次加密结果相同，nonce 未随机化")
	}
}

// TestDecryptWrongKey 错误密钥解密应失败
func TestDecryptWrongKey(t *testing.T) {
	encrypted, err := Encrypt("sk-secret", "correct-key")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if _, err := Decrypt(encrypted, "wrong-key"); err == nil {
		t.Fatal("使用错误密钥解密应当失败")
	}
}

// TestEmptySecret 空密钥应返回明确错误
func TestEmptySecret(t *testing.T) {
	if _, err := Encrypt("plaintext", ""); err != ErrEmptySecret {
		t.Fatalf("空密钥加密应返回 ErrEmptySecret: %v", err)
	}
	if _, err := Decrypt("ciphertext", ""); err != ErrEmptySecret {
		t.Fatalf("空密钥解密应返回 ErrEmptySecret: %v", err)
	}
}

// TestEmptyPlaintext 空明文加密/解密应返回空字符串（本地无鉴权服务场景）
func TestEmptyPlaintext(t *testing.T) {
	encrypted, err := Encrypt("", "secret")
	if err != nil {
		t.Fatalf("空明文加密不应报错: %v", err)
	}
	if encrypted != "" {
		t.Fatalf("空明文密文应为空: %q", encrypted)
	}

	decrypted, err := Decrypt("", "secret")
	if err != nil {
		t.Fatalf("空密文解密不应报错: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("空密文解密结果应为空: %q", decrypted)
	}
}

// TestDecryptCorrupted 损坏密文解密应失败
func TestDecryptCorrupted(t *testing.T) {
	if _, err := Decrypt("not-valid-base64!!!", "secret"); err == nil {
		t.Fatal("非法 base64 密文解密应当失败")
	}
	if _, err := Decrypt("aGVsbG8=", "secret"); err == nil {
		t.Fatal("长度不足的密文解密应当失败")
	}
}
