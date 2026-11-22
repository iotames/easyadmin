package credential

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestCredentialEncryptRoundTrip(t *testing.T) {
	raw, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	key, err := ParseKey(raw)
	if err != nil {
		t.Fatalf("解析密钥失败: %v", err)
	}
	enc, err := Encrypt("p@ss 中文 123", key)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if !strings.HasPrefix(enc, cipherPrefix) {
		t.Fatalf("密文缺少版本前缀: %q", enc)
	}
	if strings.Contains(enc, "p@ss") {
		t.Fatalf("密文疑似包含明文: %q", enc)
	}
	got, err := Decrypt(enc, key)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if got != "p@ss 中文 123" {
		t.Fatalf("解密结果不一致: %q", got)
	}
	// 同一明文两次加密必须因随机 nonce 而不同。
	enc2, _ := Encrypt("p@ss 中文 123", key)
	if enc == enc2 {
		t.Fatal("两次加密结果相同：随机数没有生效")
	}
}

func TestCredentialParseKeyInvalid(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want error
	}{
		{name: "空值", raw: "", want: ErrKeyMissing},
		{name: "非base64", raw: "not base64!", want: ErrKeyInvalid},
		{name: "长度不足", raw: base64.StdEncoding.EncodeToString([]byte("short")), want: ErrKeyInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseKey(tc.raw)
			if !errors.Is(err, tc.want) {
				t.Fatalf("期望 %v，实际 %v", tc.want, err)
			}
		})
	}
}

func TestCredentialDecryptWrongKey(t *testing.T) {
	key1, _ := ParseKey(mustGenerateKey(t))
	key2, _ := ParseKey(mustGenerateKey(t))
	enc, err := Encrypt("secret", key1)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if _, err := Decrypt(enc, key2); !errors.Is(err, ErrKeyInvalid) {
		t.Fatalf("换密钥解密应失败并归类 ErrKeyInvalid，实际 %v", err)
	}
	if _, err := Decrypt("plain-text", key1); err == nil {
		t.Fatal("缺少版本前缀的密文应被拒绝")
	}
	if got, err := Decrypt("", key1); err != nil || got != "" {
		t.Fatalf("空密文应解密为空串，实际 %q / %v", got, err)
	}
}

func mustGenerateKey(t *testing.T) string {
	t.Helper()
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	return key
}
