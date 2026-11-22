// Package credential 提供运维凭据的加密与 SSH 连接测试能力。
//
// 本包只做通用能力：AES-GCM 加解密、~/.ssh/config 别名解析、SSH 握手与认证。
// 它不认识数据库、HTTP 与业务语义，服务器记录怎么存取由装配层决定。
package credential

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 连接类型与认证方式的取值。
const (
	ConnectDirect = "direct"
	ConnectConfig = "config"

	AuthKey      = "key"
	AuthPassword = "password"
)

// 字段长度上限，与服务器记录建表脚本保持一致（表结构由 apps/system 定义）。
const (
	NameMaxLen     = 64
	HostMaxLen     = 255
	UsernameMaxLen = 64
	KeyPathMaxLen  = 500
	AliasMaxLen    = 64
	RemarkMaxLen   = 500
)

// 业务错误：上层据此映射 400 / 404 / 409，不靠错误文案判断。
var (
	ErrInvalid      = errors.New("服务器配置不合法")
	ErrNotFound     = errors.New("服务器记录不存在")
	ErrExists       = errors.New("服务器名称已被占用")
	ErrKeyMissing   = errors.New("未配置凭据加密密钥")
	ErrKeyInvalid   = errors.New("凭据加密密钥不合法")
	ErrPasswordKept = errors.New("密码模式需要凭据加密密钥")
)

// Server 是一条 SSH 服务器记录（密码字段为密文，可空）。
type Server struct {
	ID            int64
	Name          string
	ConnectType   string
	Host          string
	Port          int
	Username      string
	AuthType      string
	KeyPath       string
	PasswordEnc   string
	ConfigAlias   string
	Remark        string
	HasPassword   bool
	LastTestAt    *time.Time
	LastTestOK    bool
	LastTestError string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Input 是新增与修改的入参；Password 为明文，只在服务端使用。
// PasswordSet 区分「未提交密码」与「提交了空密码」：编辑时未提交表示保持原密码。
type Input struct {
	Name        string
	ConnectType string
	Host        string
	Port        int
	Username    string
	AuthType    string
	KeyPath     string
	ConfigAlias string
	Remark      string
	Password    string
	PasswordSet bool
}

// Resolved 是解析后的连接参数（密码为明文，只在内存中短暂存在）。
type Resolved struct {
	Host     string
	Port     int
	Username string
	AuthType string
	KeyPath  string
	Password string
	Alias    string
}

// aliasRe 是 ~/.ssh/config 别名的宽松校验：只允许字母数字与 . _ -。
var aliasRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateAlias 校验 ssh config 的 Host 别名：含空格、通配或选项字符一律拒绝。
func ValidateAlias(alias string) error {
	alias = strings.TrimSpace(alias)
	switch {
	case alias == "":
		return fmt.Errorf("%w：使用配置别名时 config_alias 不能为空，请填写 ~/.ssh/config 中的 Host 名", ErrInvalid)
	case len(alias) > AliasMaxLen:
		return fmt.Errorf("%w：配置别名过长（%d 字符），上限 %d 字符", ErrInvalid, len(alias), AliasMaxLen)
	case !aliasRe.MatchString(alias):
		return fmt.Errorf("%w：配置别名 %q 含非法字符，只允许字母、数字与 . _ -（通配符不允许，避免匹配到非预期主机）", ErrInvalid, alias)
	}
	return nil
}

// Validate 校验服务器配置，返回去掉首尾空格后的副本。
// 密码模式的加密与密钥可用性由仓储层在落库前检查，本函数只做形状校验。
func Validate(in Input) (Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.ConnectType = strings.TrimSpace(in.ConnectType)
	in.Host = strings.TrimSpace(in.Host)
	in.Username = strings.TrimSpace(in.Username)
	in.AuthType = strings.TrimSpace(in.AuthType)
	in.KeyPath = strings.TrimSpace(in.KeyPath)
	in.ConfigAlias = strings.TrimSpace(in.ConfigAlias)
	in.Remark = strings.TrimSpace(in.Remark)

	if in.Name == "" {
		return in, fmt.Errorf("%w：服务器名称不能为空，请填写一个便于识别的名称（如 erp-jump）", ErrInvalid)
	}
	if len([]rune(in.Name)) > NameMaxLen {
		return in, fmt.Errorf("%w：服务器名称过长（%d 字符），上限 %d 字符", ErrInvalid, len([]rune(in.Name)), NameMaxLen)
	}
	if len([]rune(in.Remark)) > RemarkMaxLen {
		return in, fmt.Errorf("%w：备注过长（%d 字符），上限 %d 字符", ErrInvalid, len([]rune(in.Remark)), RemarkMaxLen)
	}
	switch in.ConnectType {
	case ConnectDirect:
		if in.Host == "" {
			return in, fmt.Errorf("%w：直连模式必须填写主机地址（IP 或域名）", ErrInvalid)
		}
		if len(in.Host) > HostMaxLen {
			return in, fmt.Errorf("%w：主机地址过长（%d 字符），上限 %d 字符", ErrInvalid, len(in.Host), HostMaxLen)
		}
		if in.Port <= 0 || in.Port > 65535 {
			return in, fmt.Errorf("%w：端口 %d 不合法，取值 1~65535", ErrInvalid, in.Port)
		}
		if in.Username == "" {
			return in, fmt.Errorf("%w：直连模式必须填写登录用户名", ErrInvalid)
		}
		if len(in.Username) > UsernameMaxLen {
			return in, fmt.Errorf("%w：登录用户名过长（%d 字符），上限 %d 字符", ErrInvalid, len(in.Username), UsernameMaxLen)
		}
	case ConnectConfig:
		if err := ValidateAlias(in.ConfigAlias); err != nil {
			return in, err
		}
	default:
		return in, fmt.Errorf("%w：连接类型 %q 不可用，只能是 direct（直连）或 config（使用 ~/.ssh/config 别名）", ErrInvalid, in.ConnectType)
	}

	switch in.AuthType {
	case AuthKey:
		// 配置别名模式允许私钥路径留空：此时使用 ~/.ssh/config 的 IdentityFile。
		if in.KeyPath == "" && in.ConnectType != ConnectConfig {
			return in, fmt.Errorf("%w：密钥认证必须填写私钥路径（服务账号可读的绝对路径）", ErrInvalid)
		}
		if len(in.KeyPath) > KeyPathMaxLen {
			return in, fmt.Errorf("%w：私钥路径过长（%d 字符），上限 %d 字符", ErrInvalid, len(in.KeyPath), KeyPathMaxLen)
		}
	case AuthPassword:
		if in.Password == "" && !in.PasswordSet {
			return in, fmt.Errorf("%w：密码认证必须提交登录密码", ErrInvalid)
		}
	default:
		return in, fmt.Errorf("%w：认证方式 %q 不可用，只能是 key（私钥）或 password（密码）", ErrInvalid, in.AuthType)
	}
	return in, nil
}

// Resolve 把记录与明文密码组合成连接参数；config 模式需要先经 ResolveAlias。
func Resolve(s Server, password string) (Resolved, error) {
	out := Resolved{
		Host:     strings.TrimSpace(s.Host),
		Port:     s.Port,
		Username: strings.TrimSpace(s.Username),
		AuthType: s.AuthType,
		KeyPath:  strings.TrimSpace(s.KeyPath),
		Password: password,
		Alias:    strings.TrimSpace(s.ConfigAlias),
	}
	if s.ConnectType == ConnectConfig {
		return out, nil
	}
	if out.Port <= 0 {
		out.Port = 22
	}
	return out, nil
}

// DialAddress 返回 host:port 形式的目标地址，IPv6 主机自动加方括号。
func DialAddress(host string, port int) string {
	if port <= 0 {
		port = 22
	}
	return net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port))
}
