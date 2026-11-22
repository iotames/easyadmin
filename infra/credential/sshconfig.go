package credential

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Runner 执行外部命令并把标准输出返回给调用方；测试用替身替换真实 ssh 命令。
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultRunner 调用操作系统上的 ssh 命令。
func defaultRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return out, fmt.Errorf("ssh 命令退出码 %d: %s", exit.ExitCode(), strings.TrimSpace(string(exit.Stderr)))
		}
		return out, err
	}
	return out, nil
}

// LookupAlias 用 `ssh -G <alias>` 解析 ~/.ssh/config 别名，返回主机、端口、用户名与私钥路径。
// 只读取配置，不发起任何网络连接。
func LookupAlias(ctx context.Context, alias string, runner Runner) (Resolved, error) {
	if err := ValidateAlias(alias); err != nil {
		return Resolved{}, err
	}
	if runner == nil {
		runner = defaultRunner
	}
	if _, err := exec.LookPath("ssh"); err != nil {
		return Resolved{}, fmt.Errorf("%w：当前运行环境找不到 ssh 命令，无法解析 ~/.ssh/config 别名；请改用直连模式并填写主机、端口与用户名", ErrInvalid)
	}
	out, err := runner(ctx, "ssh", "-G", strings.TrimSpace(alias))
	if err != nil {
		return Resolved{}, fmt.Errorf("解析 ssh 配置别名 %q 失败：请确认服务账号的 ~/.ssh/config 中存在该 Host 且 ssh 命令可用: %w", alias, err)
	}
	resolved, err := ParseSSHConfigOutput(string(out))
	if err != nil {
		return Resolved{}, err
	}
	resolved.Alias = strings.TrimSpace(alias)
	// ssh -G 对未命中的别名也会成功返回：hostname 回落为别名本身、user 回落为当前登录名。
	// 这里把「回落」识别为配置未命中，避免拿别名当主机名直连后报出误导性的连接失败。
	if strings.EqualFold(strings.TrimSpace(resolved.Host), resolved.Alias) {
		return Resolved{}, &Failure{
			Stage: StageConfig, Kind: KindConfigMissing, Cause: ErrInvalid,
			Msg: fmt.Sprintf("ssh 配置中不存在与 %q 匹配的 Host 条目：请确认运行本服务的系统账号的 ~/.ssh/config 已配置该 Host（可先用该账号执行 ssh -G %s 核对输出，注意交互终端与服务进程的账号可能不同），或改用直填主机模式", alias, alias),
		}
	}
	return resolved, nil
}

// ParseSSHConfigOutput 解析 `ssh -G` 的输出。
// 该命令会输出全部生效配置（含默认值），这里只取 hostname、port、user 与第一个 identityfile。
func ParseSSHConfigOutput(output string) (Resolved, error) {
	var out Resolved
	out.Port = 22
	seenHost := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch strings.ToLower(key) {
		case "hostname":
			out.Host = value
			seenHost = true
		case "port":
			if port, err := strconv.Atoi(value); err == nil && port > 0 && port <= 65535 {
				out.Port = port
			}
		case "user":
			out.Username = value
		case "identityfile":
			if out.KeyPath == "" {
				out.KeyPath = value
			}
		}
	}
	if !seenHost || out.Host == "" {
		return Resolved{}, fmt.Errorf("ssh 配置别名解析结果缺少 hostname：请在 ~/.ssh/config 中为该 Host 配置 HostName 后重试")
	}
	if out.Username == "" {
		return Resolved{}, fmt.Errorf("ssh 配置别名解析结果缺少 user：请在 ~/.ssh/config 中为该 Host 配置 User 后重试")
	}
	return out, nil
}
