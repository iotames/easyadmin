// Package auth 是密码、会话令牌、CSRF、登录限流与授权判定的唯一入口。
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLen 是最小密码长度。
const MinPasswordLen = 8

// HashPassword 使用 bcrypt 生成密码哈希。
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword 校验密码与哈希是否匹配（常量时间比较由 bcrypt 保证）。
func VerifyPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// ValidatePasswordPolicy 校验密码强度（最小长度）。
func ValidatePasswordPolicy(pw string) error {
	if len(pw) < MinPasswordLen {
		return errors.New("密码长度至少 " + itoa(MinPasswordLen) + " 位")
	}
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
