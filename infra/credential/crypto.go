package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// cipherPrefix 标记密文格式版本：以后换算法或换字段时可以识别旧值。
const cipherPrefix = "v1:"

// keyLen 是 AES-256 的密钥长度。
const keyLen = 32

// ParseKey 解析 APP_CREDENTIAL_KEY：必须是 base64 编码的 32 字节随机值。
// 空串返回 ErrKeyMissing，长度或编码不对返回 ErrKeyInvalid。
func ParseKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w：请在 .env 或环境变量里配置 APP_CREDENTIAL_KEY（base64 编码的 32 字节随机值）后重启", ErrKeyMissing)
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%w：APP_CREDENTIAL_KEY 不是合法的 base64：请用 `openssl rand -base64 32` 生成后重新配置", ErrKeyInvalid)
	}
	if len(key) != keyLen {
		return nil, fmt.Errorf("%w：APP_CREDENTIAL_KEY 解码后是 %d 字节，必须是 32 字节（AES-256）：请重新生成后配置", ErrKeyInvalid, len(key))
	}
	return key, nil
}

// Encrypt 用 AES-256-GCM 加密明文，返回 v1:<base64(nonce|ciphertext)>。
func Encrypt(plaintext string, key []byte) (string, error) {
	if len(key) != keyLen {
		return "", fmt.Errorf("%w：加密密钥必须是 32 字节，当前 %d 字节", ErrKeyInvalid, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("初始化 AES 加密失败：%w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("初始化 AES-GCM 失败：%w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成加密随机数失败：%w", err)
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, sealed...)
	return cipherPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt 解密 Encrypt 生成的密文；格式不对、密钥不匹配或内容被篡改都返回错误。
func Decrypt(ciphertext string, key []byte) (string, error) {
	ciphertext = strings.TrimSpace(ciphertext)
	if ciphertext == "" {
		return "", nil
	}
	if len(key) != keyLen {
		return "", fmt.Errorf("%w：解密密钥必须是 32 字节，当前 %d 字节", ErrKeyInvalid, len(key))
	}
	if !strings.HasPrefix(ciphertext, cipherPrefix) {
		return "", fmt.Errorf("密码密文格式不可识别：缺少 %s 前缀，可能是旧版本或手工写入的值，请重新录入密码", cipherPrefix)
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, cipherPrefix))
	if err != nil {
		return "", fmt.Errorf("密码密文不是合法的 base64：请重新录入密码后重试: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("初始化 AES 解密失败：%w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("初始化 AES-GCM 解密失败：%w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("密码密文长度不足：无法取出随机数，请重新录入密码")
	}
	nonce, body := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("%w：密码解密失败，APP_CREDENTIAL_KEY 可能与录入时不同，请恢复原密钥或重新录入密码", ErrKeyInvalid)
	}
	return string(plain), nil
}

// GenerateKey 生成 base64 编码的 32 字节随机密钥，供运维初始化配置。
func GenerateKey() (string, error) {
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("生成随机密钥失败：%w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
