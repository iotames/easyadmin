package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// CookieName 是会话 Cookie 名。
const CookieName = "ea_session"

// NewSessionToken 生成随机会话令牌，返回 Cookie 原文与入库哈希。
// 数据库只存哈希，原文只出现在 Cookie 中。
func NewSessionToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

// HashToken 返回 token 的 SHA-256 十六进制摘要。
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
