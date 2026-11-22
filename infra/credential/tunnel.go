package credential

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// Tunnel 是一条已认证的 SSH 连接，把目标地址的 TCP 连接经跳板机转发。
// 它不监听本地端口：数据库驱动直接使用 DialContext 得到的连接。
type Tunnel struct {
	client  *ssh.Client
	timeout time.Duration
}

// NewTunnel 建立到跳板机的 SSH 连接，供后续按需转发数据库连接。
func NewTunnel(ctx context.Context, srv Server, password, knownHostsPath string, timeout time.Duration) (*Tunnel, error) {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	resolved, err := Resolve(srv, password)
	if err != nil {
		return nil, err
	}
	if srv.ConnectType == ConnectConfig {
		aliasResult, err := LookupAlias(ctx, srv.ConfigAlias, nil)
		if err != nil {
			return nil, err
		}
		resolved.Host = aliasResult.Host
		resolved.Port = aliasResult.Port
		resolved.Username = aliasResult.Username
		if resolved.KeyPath == "" && resolved.AuthType == AuthKey {
			resolved.KeyPath = aliasResult.KeyPath
		}
	}
	clientConfig, err := buildClientConfig(resolved, knownHostsPath, timeout)
	if err != nil {
		return nil, err
	}
	addr := DialAddress(resolved.Host, resolved.Port)
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("%w：建立到跳板机 %s 的 TCP 连接失败，请检查地址、端口与网络", ErrInvalid, addr)
	}
	type result struct {
		client *ssh.Client
		err    error
	}
	done := make(chan result, 1)
	go func() {
		clientConn, chans, reqs, hsErr := ssh.NewClientConn(conn, addr, clientConfig)
		if hsErr != nil {
			done <- result{err: hsErr}
			return
		}
		done <- result{client: ssh.NewClient(clientConn, chans, reqs)}
	}()
	select {
	case <-ctx.Done():
		_ = conn.Close()
		return nil, fmt.Errorf("建立 SSH 隧道被取消：%w", ctx.Err())
	case r := <-done:
		if r.err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("SSH 隧道握手失败：%s", describeHandshakeError(r.err))
		}
		return &Tunnel{client: r.client, timeout: timeout}, nil
	}
}

// DialContext 经 SSH 连接打开到目标地址的转发连接。
// x/crypto 的 Dial 不支持上下文，这里用 goroutine ＋ select 保证取消与超时生效。
func (t *Tunnel) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if t == nil || t.client == nil {
		return nil, fmt.Errorf("SSH 隧道未建立：请先确认服务器连接测试通过")
	}
	done := make(chan dialResult, 1)
	go func() {
		conn, err := t.client.Dial(network, addr)
		done <- dialResult{conn: conn, err: err}
	}()
	timer := time.NewTimer(t.timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		discardLateDial(done)
		return nil, fmt.Errorf("经 SSH 隧道连接 %s 被取消：%w", addr, ctx.Err())
	case <-timer.C:
		discardLateDial(done)
		return nil, fmt.Errorf("经 SSH 隧道连接 %s 超时（%s）：请检查跳板机到目标库的网络与目标库是否监听", addr, t.timeout)
	case r := <-done:
		if r.err != nil {
			return nil, fmt.Errorf("经 SSH 隧道连接 %s 失败：请检查跳板机到目标库的网络、端口与防火墙: %w", addr, r.err)
		}
		return r.conn, nil
	}
}

// dialResult 是异步拨号结果；成功时 conn 必须由接收方负责关闭。
type dialResult struct {
	conn net.Conn
	err  error
}

// discardLateDial 在超时/取消先返回后，回收随后才建立的连接，避免 fd 与 SSH 通道泄漏。
func discardLateDial(done <-chan dialResult) {
	go func() {
		if r := <-done; r.conn != nil {
			_ = r.conn.Close()
		}
	}()
}

// Close 关闭 SSH 连接；重复关闭不报错。
func (t *Tunnel) Close() error {
	if t == nil || t.client == nil {
		return nil
	}
	return t.client.Close()
}
