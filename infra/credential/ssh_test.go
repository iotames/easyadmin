package credential

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestParseSSHConfigOutput(t *testing.T) {
	output := strings.Join([]string{
		"host erp-jump",
		"hostname 10.0.0.8",
		"port 2202",
		"user deploy",
		"identityfile /home/deploy/.ssh/id_ed25519",
		"identityfile /home/deploy/.ssh/id_rsa",
		"",
	}, "\n")
	got, err := ParseSSHConfigOutput(output)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if got.Host != "10.0.0.8" || got.Port != 2202 || got.Username != "deploy" {
		t.Fatalf("解析结果不符: %+v", got)
	}
	if got.KeyPath != "/home/deploy/.ssh/id_ed25519" {
		t.Fatalf("应取第一个 identityfile，实际 %q", got.KeyPath)
	}
	if _, err := ParseSSHConfigOutput("port 22\nuser root\n"); err == nil {
		t.Fatal("缺少 hostname 时应报错")
	}
}

func TestValidateServerInput(t *testing.T) {
	base := Input{Name: "jump", ConnectType: ConnectDirect, Host: "10.0.0.8", Port: 22, Username: "deploy", AuthType: AuthKey, KeyPath: "/home/deploy/.ssh/id_ed25519"}
	if _, err := Validate(base); err != nil {
		t.Fatalf("合法配置被拒绝: %v", err)
	}
	cases := []struct {
		name string
		in   Input
	}{
		{name: "缺名称", in: func() Input { v := base; v.Name = ""; return v }()},
		{name: "缺主机", in: func() Input { v := base; v.Host = ""; return v }()},
		{name: "端口越界", in: func() Input { v := base; v.Port = 70000; return v }()},
		{name: "缺用户名", in: func() Input { v := base; v.Username = ""; return v }()},
		{name: "密钥模式缺路径", in: func() Input { v := base; v.KeyPath = ""; return v }()},
		{name: "密码模式缺密码", in: func() Input { v := base; v.AuthType = AuthPassword; return v }()},
		{name: "别名含通配符", in: func() Input {
			v := base
			v.ConnectType = ConnectConfig
			v.ConfigAlias = "erp-*"
			v.KeyPath = ""
			return v
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Validate(tc.in); !errors.Is(err, ErrInvalid) {
				t.Fatalf("应返回 ErrInvalid，实际 %v", err)
			}
		})
	}
	// 配置别名模式下允许私钥路径留空（由 ssh config 的 IdentityFile 提供）。
	cfg := base
	cfg.ConnectType = ConnectConfig
	cfg.ConfigAlias = "erp-jump"
	cfg.KeyPath = ""
	if _, err := Validate(cfg); err != nil {
		t.Fatalf("配置别名模式应允许留空私钥路径: %v", err)
	}
}

func TestSSHPasswordAuthSuccess(t *testing.T) {
	srv := startFakeSSH(t, fakeSSHOptions{password: "s3cret"})
	known := writeKnownHosts(t, srv.addr, srv.hostKey.PublicKey())

	result, err := Test(context.Background(), TestOptions{
		Server: Server{
			Name: "jump", ConnectType: ConnectDirect, Host: "127.0.0.1", Port: srv.port,
			Username: "deploy", AuthType: AuthPassword,
		},
		Password:       "s3cret",
		KnownHostsPath: known,
		Timeout:        5 * time.Second,
	})
	if err != nil {
		t.Fatalf("密码认证应成功，实际 %v（分段: %+v）", err, result.Steps)
	}
	if !result.OK || len(result.Steps) != 3 {
		t.Fatalf("结果不符: %+v", result)
	}
}

func TestSSHKeyAuthSuccess(t *testing.T) {
	clientKey, clientPriv := newKeyPair(t)
	srv := startFakeSSH(t, fakeSSHOptions{authorized: clientKey.PublicKey()})
	known := writeKnownHosts(t, srv.addr, srv.hostKey.PublicKey())
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	writePrivateKey(t, keyPath, clientPriv)

	result, err := Test(context.Background(), TestOptions{
		Server: Server{
			Name: "jump", ConnectType: ConnectDirect, Host: "127.0.0.1", Port: srv.port,
			Username: "deploy", AuthType: AuthKey, KeyPath: keyPath,
		},
		KnownHostsPath: known,
		Timeout:        5 * time.Second,
	})
	if err != nil {
		t.Fatalf("密钥认证应成功，实际 %v（分段: %+v）", err, result.Steps)
	}
	if !result.OK {
		t.Fatalf("结果不符: %+v", result)
	}
}

func TestSSHAuthFailedClassification(t *testing.T) {
	srv := startFakeSSH(t, fakeSSHOptions{password: "right"})
	known := writeKnownHosts(t, srv.addr, srv.hostKey.PublicKey())
	_, err := Test(context.Background(), TestOptions{
		Server: Server{
			Name: "jump", ConnectType: ConnectDirect, Host: "127.0.0.1", Port: srv.port,
			Username: "deploy", AuthType: AuthPassword,
		},
		Password:       "wrong",
		KnownHostsPath: known,
		Timeout:        5 * time.Second,
	})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Kind != KindAuthFailed {
		t.Fatalf("密码错误应归类为认证失败，实际 %v", err)
	}
}

func TestSSHUnknownHostKey(t *testing.T) {
	srv := startFakeSSH(t, fakeSSHOptions{password: "s3cret"})
	// known_hosts 里只有另一台主机的记录：目标主机属于「未登记」而非「密钥不一致」。
	known := writeKnownHosts(t, "other.example:22", newSigner(t).PublicKey())
	_, err := Test(context.Background(), TestOptions{
		Server: Server{
			Name: "jump", ConnectType: ConnectDirect, Host: "127.0.0.1", Port: srv.port,
			Username: "deploy", AuthType: AuthPassword,
		},
		Password:       "s3cret",
		KnownHostsPath: known,
		Timeout:        5 * time.Second,
	})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Kind != KindHostKeyUnknown {
		t.Fatalf("未知主机密钥应被拒绝，实际 %v", err)
	}
}

func TestSSHKnownHostsAbsent(t *testing.T) {
	srv := startFakeSSH(t, fakeSSHOptions{password: "s3cret"})
	_, err := Test(context.Background(), TestOptions{
		Server: Server{
			Name: "jump", ConnectType: ConnectDirect, Host: "127.0.0.1", Port: srv.port,
			Username: "deploy", AuthType: AuthPassword,
		},
		Password:       "s3cret",
		KnownHostsPath: filepath.Join(t.TempDir(), "missing_known_hosts"),
		Timeout:        5 * time.Second,
	})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Kind != KindKnownHostsAbsent {
		t.Fatalf("known_hosts 缺失应被明确拒绝，实际 %v", err)
	}
}

func TestSSHUnreachableClassification(t *testing.T) {
	// 先占一个端口再关闭，得到本机几乎必然拒绝连接的地址。
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	known := writeKnownHosts(t, "127.0.0.1:"+fmt.Sprint(port), newSigner(t).PublicKey())
	_, err = Test(context.Background(), TestOptions{
		Server: Server{
			Name: "jump", ConnectType: ConnectDirect, Host: "127.0.0.1", Port: port,
			Username: "deploy", AuthType: AuthPassword,
		},
		Password:       "x",
		KnownHostsPath: known,
		Timeout:        2 * time.Second,
	})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Kind != KindUnreachable {
		t.Fatalf("不可达地址应归类为连接失败，实际 %v", err)
	}
}

func TestLookupAliasUsesRunner(t *testing.T) {
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "ssh" || len(args) != 2 || args[0] != "-G" || args[1] != "erp-jump" {
			return nil, fmt.Errorf("意外的命令: %s %v", name, args)
		}
		return []byte("hostname 10.0.0.9\nport 2222\nuser deploy\nidentityfile /home/deploy/.ssh/id_ed25519\n"), nil
	}
	got, err := LookupAlias(context.Background(), "erp-jump", runner)
	if err != nil {
		t.Fatalf("解析别名失败: %v", err)
	}
	if got.Host != "10.0.0.9" || got.Port != 2222 || got.Username != "deploy" || got.Alias != "erp-jump" {
		t.Fatalf("别名解析结果不符: %+v", got)
	}
	if _, err := LookupAlias(context.Background(), "../etc", runner); !errors.Is(err, ErrInvalid) {
		t.Fatalf("非法别名应被拒绝，实际 %v", err)
	}
}

func TestLookupAliasDetectsUnmatchedConfig(t *testing.T) {
	// ssh -G 对未命中的别名也返回成功，hostname 回落为别名本身、user 回落为当前登录名：
	// 必须识别为「配置未命中」，不能拿别名当主机名去直连。
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("hostname erpalias\nport 22\nuser deploy\n"), nil
	}
	_, err := LookupAlias(context.Background(), "erpalias", runner)
	if err == nil {
		t.Fatal("hostname 回落为别名本身时应报配置未命中错误")
	}
	var failure *Failure
	if !errors.As(err, &failure) || failure.Kind != KindConfigMissing {
		t.Fatalf("应归类为 config_missing，实际 %v", err)
	}
	if !strings.Contains(err.Error(), "erpalias") || !strings.Contains(err.Error(), "ssh -G") {
		t.Fatalf("错误信息要写明别名与核对命令，实际 %v", err)
	}
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("本机无法确定主目录：%v", err)
	}
	cases := []struct{ in, want string }{
		{"~", home},
		{"~/.ssh/erpalias.pri", filepath.Join(home, ".ssh", "erpalias.pri")},
		{"/root/.ssh/erpalias.pri", "/root/.ssh/erpalias.pri"},
		{"", ""},
		{"usr~/.ssh/a", "usr~/.ssh/a"},
	}
	for _, c := range cases {
		got, err := expandTilde(c.in)
		if err != nil {
			t.Fatalf("expandTilde(%q) 意外出错：%v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("expandTilde(%q) = %q，期望 %q", c.in, got, c.want)
		}
	}
}

// ---------- 测试用 SSH 替身 ----------

type fakeSSHOptions struct {
	password   string
	authorized ssh.PublicKey
}

type fakeSSH struct {
	addr    string
	port    int
	hostKey ssh.Signer
}

// startFakeSSH 在回环地址起一个只做认证的 SSH 服务端，不提供任何会话能力。
func startFakeSSH(t *testing.T, opts fakeSSHOptions) fakeSSH {
	t.Helper()
	hostKey := newSigner(t)
	config := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if opts.password != "" && string(pass) == opts.password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected")
		},
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if opts.authorized != nil && string(key.Marshal()) == string(opts.authorized.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("key rejected")
		},
	}
	config.AddHostKey(hostKey)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("启动测试 SSH 服务失败: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, chans, reqs, err := ssh.NewServerConn(c, config)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for ch := range chans {
					_ = ch.Reject(ssh.Prohibited, "test server")
				}
			}(conn)
		}
	}()
	addr := ln.Addr().String()
	return fakeSSH{addr: addr, port: ln.Addr().(*net.TCPAddr).Port, hostKey: hostKey}
}

func newSigner(t *testing.T) ssh.Signer {
	t.Helper()
	signer, _ := newKeyPair(t)
	return signer
}

func newKeyPair(t *testing.T) (ssh.Signer, ed25519.PrivateKey) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成密钥失败: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("构造签名器失败: %v", err)
	}
	return signer, priv
}

func writeKnownHosts(t *testing.T, addr string, key ssh.PublicKey) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "known_hosts")
	line := knownHostsAddress(addr) + " " + strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))) + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatalf("写入 known_hosts 失败: %v", err)
	}
	return path
}

// knownHostsAddress 把 host:port 归一成 [host]:port（非 22 端口必须带方括号）。
func knownHostsAddress(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if port == "22" {
		return host
	}
	return "[" + host + "]:" + port
}

func writePrivateKey(t *testing.T, path string, priv ed25519.PrivateKey) {
	t.Helper()
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("编码私钥失败: %v", err)
	}
	pemBytes := pem.EncodeToMemory(block)
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("写入私钥失败: %v", err)
	}
}
