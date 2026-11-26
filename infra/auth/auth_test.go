package auth

import (
	"testing"
	"time"
)

func TestPasswordHashAndVerify(t *testing.T) {
	h, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(h, "password123") {
		t.Fatal("正确密码应通过")
	}
	if VerifyPassword(h, "wrong") {
		t.Fatal("错误密码应失败")
	}
	if err := ValidatePasswordPolicy("short"); err == nil {
		t.Fatal("短密码应被拒绝")
	}
	if err := ValidatePasswordPolicy("longenough"); err != nil {
		t.Fatal(err)
	}
}

func TestSessionTokenAndCSRF(t *testing.T) {
	raw, hash, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == "" || raw == hash {
		t.Fatalf("令牌异常 raw=%q hash=%q", raw, hash)
	}
	if HashToken(raw) != hash {
		t.Fatal("HashToken 不一致")
	}
	c := NewCSRF()
	tok := c.Token(hash)
	if !c.Validate(hash, tok) {
		t.Fatal("CSRF 校验应通过")
	}
	if c.Validate(hash, "bad") {
		t.Fatal("错误 CSRF 应失败")
	}
	if c.Validate("other", tok) {
		t.Fatal("跨会话 CSRF 应失败")
	}
}

func TestRateLimiter(t *testing.T) {
	l := NewRateLimiter(2, time.Minute)
	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("第 %d 次应放行", i+1)
		}
	}
	if ok, retry := l.Allow("k"); ok || retry <= 0 {
		t.Fatal("超过上限应拒绝并给出重试秒数")
	}
	l.Reset("k")
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("重置后应放行")
	}
}

func TestAuthorize(t *testing.T) {
	if !Has([]string{"*"}, "system.user.read") {
		t.Fatal("通配权限应通过")
	}
	if !Has([]string{"system.user.read"}, "system.user.read") {
		t.Fatal("精确权限应通过")
	}
	if Has([]string{"a.b"}, "c.d") {
		t.Fatal("无权限应拒绝")
	}
	if !Has(nil, "") {
		t.Fatal("空需求应通过")
	}
}
