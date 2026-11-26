package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// CSRF 以服务端随机密钥对会话 ID 做 HMAC，生成与会话绑定、无需落库的 CSRF token。
type CSRF struct {
	secret []byte
}

// NewCSRF 生成随机密钥。单实例进程内密钥即可满足会话绑定语义。
func NewCSRF() *CSRF {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: 无法生成 CSRF 密钥: " + err.Error())
	}
	return &CSRF{secret: b}
}

// Token 返回与会话绑定的 CSRF token。
func (c *CSRF) Token(sessionID string) string {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

// Validate 常量时间校验 CSRF token。
func (c *CSRF) Validate(sessionID, token string) bool {
	if sessionID == "" || token == "" {
		return false
	}
	expected := c.Token(sessionID)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(token)) == 1
}
