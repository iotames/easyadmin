package credential

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// 连接测试的阶段名与错误分类：调用方据此区分「跳板不可达」「主机密钥未知」「认证失败」。
const (
	StageConfig       = "ssh_config"
	StageTCP          = "ssh_tcp"
	StageHandshake    = "ssh_handshake"
	StageDone         = "done"
	TestStepConfig    = "解析配置别名"
	TestStepTCP       = "建立 TCP 连接"
	TestStepHandshake = "SSH 握手与认证"
)

// Failure 是带阶段与分类的测试失败，便于接口给出分段结果。
type Failure struct {
	Stage string
	Kind  string
	Msg   string
	Cause error
}

func (e *Failure) Error() string {
	if e.Cause != nil {
		return e.Msg + ": " + e.Cause.Error()
	}
	return e.Msg
}

func (e *Failure) Unwrap() error { return e.Cause }

// 失败分类。
const (
	KindConfigMissing    = "config_missing"
	KindUnreachable      = "unreachable"
	KindHostKeyUnknown   = "host_key_unknown"
	KindHostKeyMismatch  = "host_key_mismatch"
	KindAuthFailed       = "auth_failed"
	KindKeyUnreadable    = "key_unreadable"
	KindKeyPassphrase    = "key_passphrase_unsupported"
	KindKnownHostsAbsent = "known_hosts_absent"
)

// Step 是分段测试的一步结果。
type Step struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
	DurationMs int64  `json:"duration_ms"`
}

// TestResult 是一次连接测试的分段结论。
type TestResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Steps []Step `json:"steps"`
}

// TestOptions 是连接测试的入参。
type TestOptions struct {
	Server         Server
	Password       string
	KnownHostsPath string
	Timeout        time.Duration
	Runner         Runner
}

// Test 只做「解析配置 → TCP 连接 → SSH 握手与认证」，不执行任何远程命令。
func Test(ctx context.Context, opts TestOptions) (TestResult, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	result := TestResult{Steps: make([]Step, 0, 3)}
	srv := opts.Server

	resolved, err := Resolve(srv, opts.Password)
	if err != nil {
		return fail(result, StageConfig, KindConfigMissing, err.Error(), err)
	}
	start := time.Now()
	if srv.ConnectType == ConnectConfig {
		aliasResult, err := LookupAlias(ctx, srv.ConfigAlias, opts.Runner)
		if err != nil {
			result.Steps = append(result.Steps, Step{Name: TestStepConfig, Status: "fail", Detail: err.Error(), DurationMs: time.Since(start).Milliseconds()})
			return fail(result, StageConfig, KindConfigMissing, err.Error(), err)
		}
		resolved.Host = aliasResult.Host
		resolved.Port = aliasResult.Port
		resolved.Username = aliasResult.Username
		if resolved.KeyPath == "" && resolved.AuthType == AuthKey {
			resolved.KeyPath = aliasResult.KeyPath
		}
		detail := fmt.Sprintf("别名 %s → %s@%s:%d", aliasResult.Alias, resolved.Username, resolved.Host, resolved.Port)
		result.Steps = append(result.Steps, Step{Name: TestStepConfig, Status: "ok", Detail: detail, DurationMs: time.Since(start).Milliseconds()})
	} else {
		result.Steps = append(result.Steps, Step{Name: TestStepConfig, Status: "skip", Detail: "直连模式，无需解析别名", DurationMs: 0})
	}

	addr := DialAddress(resolved.Host, resolved.Port)
	start = time.Now()
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.Steps = append(result.Steps, Step{Name: TestStepTCP, Status: "fail", Detail: describeNetError(addr, err), DurationMs: time.Since(start).Milliseconds()})
		return fail(result, StageTCP, KindUnreachable, fmt.Sprintf("连接 %s 失败：%s", addr, describeNetError(addr, err)), err)
	}
	result.Steps = append(result.Steps, Step{Name: TestStepTCP, Status: "ok", Detail: "已建立到 " + addr + " 的 TCP 连接", DurationMs: time.Since(start).Milliseconds()})

	start = time.Now()
	clientConfig, err := buildClientConfig(resolved, opts.KnownHostsPath, timeout)
	if err != nil {
		result.Steps = append(result.Steps, Step{Name: TestStepHandshake, Status: "fail", Detail: err.Error(), DurationMs: time.Since(start).Milliseconds()})
		return fail(result, StageHandshake, classifyBuildError(err), err.Error(), err)
	}
	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type handshakeResult struct {
		err error
	}
	done := make(chan handshakeResult, 1)
	go func() {
		clientConn, _, _, hsErr := ssh.NewClientConn(conn, addr, clientConfig)
		if hsErr == nil && clientConn != nil {
			_ = clientConn.Close()
		}
		done <- handshakeResult{err: hsErr}
	}()
	var hsErr error
	select {
	case <-handshakeCtx.Done():
		_ = conn.Close()
		hsErr = fmt.Errorf("SSH 握手在 %s 内未完成：请检查跳板机负载、网络与防火墙", timeout)
	case r := <-done:
		hsErr = r.err
		_ = conn.Close()
	}
	if hsErr != nil {
		detail := describeHandshakeError(hsErr)
		result.Steps = append(result.Steps, Step{Name: TestStepHandshake, Status: "fail", Detail: detail, DurationMs: time.Since(start).Milliseconds()})
		return fail(result, StageHandshake, classifyHandshakeError(hsErr), detail, hsErr)
	}
	result.Steps = append(result.Steps, Step{Name: TestStepHandshake, Status: "ok", Detail: fmt.Sprintf("认证成功：%s@%s:%d", resolved.Username, resolved.Host, resolved.Port), DurationMs: time.Since(start).Milliseconds()})
	result.OK = true
	return result, nil
}

// fail 统一填充失败结果并返回 Failure。
func fail(result TestResult, stage, kind, msg string, cause error) (TestResult, error) {
	result.OK = false
	result.Error = msg
	return result, &Failure{Stage: stage, Kind: kind, Msg: msg, Cause: cause}
}

// expandTilde 把路径开头的 ~ 展开为服务账号主目录：OpenSSH 会展开 ~，Go 的文件读取不会，
// 因此服务端在此兜底，用户按 ssh 习惯填 ~/.ssh/key 也生效。
func expandTilde(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

// buildClientConfig 组装 SSH 客户端配置：认证方式与主机密钥校验。
func buildClientConfig(resolved Resolved, knownHostsPath string, timeout time.Duration) (*ssh.ClientConfig, error) {
	auths, err := buildAuth(resolved)
	if err != nil {
		return nil, err
	}
	callback, err := hostKeyCallback(knownHostsPath)
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            resolved.Username,
		Auth:            auths,
		HostKeyCallback: callback,
		Timeout:         timeout,
	}, nil
}

// buildAuth 按认证方式构造认证方法；私钥口令暂不支持，给出明确错误。
func buildAuth(resolved Resolved) ([]ssh.AuthMethod, error) {
	switch resolved.AuthType {
	case AuthPassword:
		if resolved.Password == "" {
			return nil, fmt.Errorf("密码认证缺少密码：请重新录入密码后测试（密码只在服务端解密使用，不回显）")
		}
		return []ssh.AuthMethod{ssh.Password(resolved.Password)}, nil
	case AuthKey:
		if resolved.KeyPath == "" {
			return nil, fmt.Errorf("密钥认证缺少私钥路径：请填写服务账号可读的私钥文件路径")
		}
		keyPath, err := expandTilde(resolved.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("%w：无法定位服务账号主目录，无法展开私钥路径 %s；请直接填写绝对路径", errKeyUnreadable, resolved.KeyPath)
		}
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("%w：读取私钥 %s 失败，请确认路径存在且服务账号有读权限", errKeyUnreadable, keyPath)
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			var passErr *ssh.PassphraseMissingError
			if errors.As(err, &passErr) {
				return nil, fmt.Errorf("%w：私钥 %s 带口令，当前版本不支持带口令的私钥，请改用免口令私钥或密码认证", errKeyPassphrase, resolved.KeyPath)
			}
			return nil, fmt.Errorf("%w：私钥 %s 无法解析（%v），请确认是 OpenSSH 私钥格式", errKeyUnreadable, resolved.KeyPath, err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, fmt.Errorf("认证方式 %q 不可用，只能是 key 或 password", resolved.AuthType)
	}
}

// 内部哨兵错误，用于把构造阶段的失败归类。
var (
	errKeyUnreadable    = errors.New("私钥不可读")
	errKeyPassphrase    = errors.New("私钥带口令")
	errKnownHostsAbsent = errors.New("主机密钥文件不存在")
)

// hostKeyCallback 加载 known_hosts 并返回严格校验回调；文件不存在时明确报错，不静默跳过校验。
func hostKeyCallback(path string) (ssh.HostKeyCallback, error) {
	resolvedPath := strings.TrimSpace(path)
	if resolvedPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("无法确定服务账号的主目录，无法定位默认 known_hosts：请在 APP_SSH_KNOWN_HOSTS 指定主机密钥文件路径后重试: %w", err)
		}
		resolvedPath = filepath.Join(home, ".ssh", "known_hosts")
	}
	if _, err := os.Stat(resolvedPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w：主机密钥文件 %s 不存在。请运维把跳板机公钥写入该文件（ssh-keyscan 的结果需人工核对指纹）后重试；本系统不会自动信任未知主机", errKnownHostsAbsent, resolvedPath)
		}
		return nil, fmt.Errorf("读取主机密钥文件 %s 失败：请确认服务账号有读权限: %w", resolvedPath, err)
	}
	callback, err := knownhosts.New(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("解析主机密钥文件 %s 失败：请检查文件格式（ssh-keyscan 输出）: %w", resolvedPath, err)
	}
	return callback, nil
}

// classifyBuildError 把构造阶段的错误归到对外分类。
func classifyBuildError(err error) string {
	switch {
	case errors.Is(err, errKnownHostsAbsent):
		return KindKnownHostsAbsent
	case errors.Is(err, errKeyUnreadable):
		return KindKeyUnreadable
	case errors.Is(err, errKeyPassphrase):
		return KindKeyPassphrase
	default:
		return KindAuthFailed
	}
}

// describeNetError 把网络错误转成可执行的中文提示。
func describeNetError(addr string, err error) string {
	if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
		return fmt.Sprintf("连接 %s 超时：请检查跳板机地址、端口与防火墙/安全组是否放行", addr)
	}
	return fmt.Sprintf("连接 %s 失败：请检查地址、端口、网络连通性与防火墙设置", addr)
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// classifyHandshakeError 把握手/认证错误归类。
func classifyHandshakeError(err error) string {
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) == 0 {
			return KindHostKeyUnknown
		}
		return KindHostKeyMismatch
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unable to authenticate") || strings.Contains(msg, "no supported methods") {
		return KindAuthFailed
	}
	if strings.Contains(msg, "handshake failed") {
		return KindHostKeyMismatch
	}
	return KindAuthFailed
}

// describeHandshakeError 把握手错误转成带下一步动作的中文提示。
func describeHandshakeError(err error) string {
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) == 0 {
			return "主机密钥未登记：请运维核对下方指纹后把该主机加入 known_hosts，再重试（本系统不会自动信任未知主机）"
		}
		return "主机密钥与 known_hosts 记录不一致：可能是跳板机重装或中间人攻击，请运维核对指纹后再更新 known_hosts"
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unable to authenticate") || strings.Contains(msg, "no supported methods") {
		return "SSH 认证失败：请检查用户名、私钥/密码是否正确，以及服务账号是否被允许登录"
	}
	return "SSH 握手失败：请检查协议版本、跳板机配置与网络链路"
}

// HostFingerprint 返回未知主机在握手时提供的指纹，用于运维人工核对。
// 只有握手因主机密钥失败时才取得到值；其余情况返回空串。
func HostFingerprint(err error) string {
	var keyErr *knownhosts.KeyError
	if !errors.As(err, &keyErr) {
		return ""
	}
	if len(keyErr.Want) == 0 {
		// knownhosts 在 Want 为空时把实际看到的密钥放在 Err 里；这里无法直接取到，
		// 由调用方在测试结果里提示运维用 ssh-keyscan 获取并核对。
		return ""
	}
	return ssh.FingerprintSHA256(keyErr.Want[0].Key)
}
