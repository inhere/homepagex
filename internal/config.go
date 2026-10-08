package internal

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Mode       string `yaml:"mode" json:"mode"`               // debug or release
	Port       string `yaml:"port" json:"port"`               // 监听端口
	SessionTTL string `yaml:"session_ttl" json:"session_ttl"` // 会话有效期，如 "2h", "30m"
	// CookieSecure 会话 cookie 的 Secure 属性策略：
	// auto（默认，请求本身是 HTTPS 才带 Secure）/ true（始终带）/ false（从不带）
	CookieSecure string `yaml:"cookie_secure" json:"cookie_secure"`
}

// cookie_secure 的三种取值（归一化后）
const (
	cookieSecureAuto  = "auto"
	cookieSecureTrue  = "true"
	cookieSecureFalse = "false"
)

// NavItem 导航项
type NavItem struct {
	Name string `yaml:"name" json:"name"`
	Icon string `yaml:"icon" json:"icon"`
	URL  string `yaml:"url" json:"url"`
	// Mode: 空值/yaml 加载 YAML 页面；iframe 在骨架内嵌入；page 使用普通链接。
	Mode   string `yaml:"mode,omitempty" json:"mode,omitempty"`
	Target string `yaml:"target,omitempty" json:"target,omitempty"`
}

// PageDefaults 页面默认配置
type PageDefaults struct {
	Title    string `yaml:"title" json:"title"`
	Subtitle string `yaml:"subtitle" json:"subtitle"`
	Logo     string `yaml:"logo" json:"logo"`
	Header   string `yaml:"header" json:"header"`
	Footer   string `yaml:"footer" json:"footer"`
	Theme    string `yaml:"theme" json:"theme"`
	Color    string `yaml:"color" json:"color"`
	Style    string `yaml:"style" json:"style"`
	Columns  string `yaml:"columns" json:"columns"`
}

// Config 应用主配置
type Config struct {
	Server      ServerConfig `yaml:"server"`
	PagesDir    string       `yaml:"pages_dir"`
	FrontendDir string       `yaml:"frontend_dir"`
	// StaticDir 为内嵌/磁盘前端补充外部 HTML、JSON 等文件；为空时不启用。
	StaticDir string `yaml:"static_dir"`
	// 图标 CDN 配置 see https://dashboardicons.com/ 搜索
	IconsCDN map[string]string `yaml:"icons_cdn"`
	// IconsDir 图标缓存目录。默认「配置文件所在目录/icons-cache」，
	// 离线部署时指向一个可写的持久化目录即可；不会写进内嵌的前端资源里。
	IconsDir string `yaml:"icons_dir"`
	// IconsRemote 是否允许从 CDN 下载缺失的图标（默认 true）。
	// 设为 false 时完全不访问外网，缓存未命中的图标直接返回 404。
	IconsRemote bool `yaml:"icons_remote"`
	// basic 认证配置，格式：user:pass@path:perm,path2:perm2
	Auths []string `yaml:"auths"`
	// deny 硬拒绝路径：任何人都不能访问，用于临时下线页面等
	Deny []string `yaml:"deny"`
	// 页面默认配置
	PageDefaults PageDefaults `yaml:"page_defaults"`
	PageNavs     []NavItem    `yaml:"page_navs"`

	// 解析后的认证配置，key 为用户名（"" 表示匿名）
	parsedAuths map[string]*AuthConfig
	// guestRules 匿名基线中「允许」的规则（不含 `!` 认证墙）
	guestRules []*pathRule
	// guestDenyRules `!` 认证墙：仅对匿名生效，登录后即越过
	guestDenyRules []*pathRule
	// hardDenyRules 顶层 deny：对所有人硬拒绝
	hardDenyRules []*pathRule
}

// AuthConfig 一个解析后的用户认证配置
type AuthConfig struct {
	Username string
	Password string
	// Rules 归一化后的匹配规则
	Rules []*pathRule
}

// IsValid 是否有效
func (c *AuthConfig) IsValid() bool {
	return c.Username != "" || c.Password != "" || len(c.Rules) > 0
}

func newDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:       "8090",
			SessionTTL: "2h",
		},
		Auths:       []string{"@*"}, // 所有路径可访问，无需认证
		PagesDir:    "./pages",
		FrontendDir: "./frontend/build",
		IconsDir:    "./icons-cache",
		IconsRemote: true,
	}
}

// DefaultConfig 返回一份内置的默认配置（不读文件），用于配置文件缺失时的兜底。
func DefaultConfig() *Config {
	config := newDefaultConfig()
	// 默认 auths 是写死的 "@*"，parseAuths 不可能失败
	_ = config.parseAuths()
	return config
}

// LoadConfig 从 YAML 文件加载配置。
//
// 返回约定：
//   - 文件不存在/读不到：返回「默认配置 + err」，调用方可选择兜底继续运行
//   - 文件存在但解析或校验失败：返回「nil + err」，调用方应当直接失败
//     （否则会带着一份与用户预期不符的配置静默启动）
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), fmt.Errorf("failed to read config file: %w", err)
	}

	// 加载默认配置后再叠加文件内容
	config := newDefaultConfig()
	if err = yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	if config.Server.Port == "" {
		config.Server.Port = "8090"
	}
	if config.Server.SessionTTL == "" {
		config.Server.SessionTTL = "2h"
	}
	if config.PagesDir == "" {
		config.PagesDir = "./pages"
	}
	if config.FrontendDir == "" {
		config.FrontendDir = "./frontend/build"
	}
	if config.IconsDir == "" {
		config.IconsDir = "./icons-cache"
	}

	// 归一化并校验 cookie_secure：写错就启动失败，避免静默回退成另一种语义
	mode, ok := normalizeCookieSecure(config.Server.CookieSecure)
	if !ok {
		return nil, fmt.Errorf("invalid server.cookie_secure %q, expect auto|true|false", config.Server.CookieSecure)
	}
	config.Server.CookieSecure = mode

	if err = config.parseAuths(); err != nil {
		return nil, fmt.Errorf("failed to parse auths: %w", err)
	}
	return config, nil
}

// ResolveRelativeDirs 把各资源目录的相对路径按「配置文件所在目录」解析。
//
// 全局配置（~/.config/homepagex/config.yaml）会在任意工作目录下被加载，
// 若仍以进程 CWD 为基准，./pages 就会指向别处 —— 换个目录启动就找不到页面了。
func (c *Config) ResolveRelativeDirs(configPath string) error {
	absFile, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to resolve config path %q: %w", configPath, err)
	}

	baseDir := filepath.Dir(absFile)
	for _, dir := range []*string{&c.PagesDir, &c.FrontendDir, &c.StaticDir, &c.IconsDir} {
		if *dir == "" || filepath.IsAbs(*dir) {
			continue
		}
		*dir = filepath.Join(baseDir, *dir)
	}
	return nil
}

// IconCacheDir 返回图标缓存目录。
//
// 优先用 icons_dir；没配置时回落到 frontend_dir/icons-local
// （保持「直接构造 Config」的历史行为），正常经 LoadConfig + ResolveRelativeDirs
// 加载的配置一定有值。
func (c *Config) IconCacheDir() string {
	if c.IconsDir != "" {
		return c.IconsDir
	}
	return filepath.Join(c.FrontendDir, IconLocalPrefix)
}

// parseAuths 解析 auths / deny 配置，产出归一化规则。
//
// 规则文本：
//   - 用户规则：user:pass@path:perm,path2:perm2（perm 省略时默认 ro）
//   - 匿名规则：@path:perm，即用户名为空，构成所有人的匿名基线
//   - `!path`：认证墙，仅对匿名生效（登录后即可见）
//   - `:no`：显式拒绝
//
// 解析失败返回 error，由 LoadConfig 直接失败 —— 配置错误不应被静默降级成另一种语义。
func (c *Config) parseAuths() error {
	auths := make(map[string]*AuthConfig)

	for _, auth := range c.Auths {
		if strings.TrimSpace(auth) == "" {
			continue
		}

		credStr, pathStr, ok := strings.Cut(auth, "@")
		if !ok {
			return fmt.Errorf("invalid auth rule %q: expect format user:pass@path:perm", auth)
		}

		ac := &AuthConfig{}
		if credStr != "" {
			if idx := strings.Index(credStr, ":"); idx == -1 {
				ac.Username = credStr
			} else {
				ac.Username = credStr[:idx]
				ac.Password = credStr[idx+1:]
			}
		}

		for token := range strings.SplitSeq(pathStr, ",") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}

			// `!path` 认证墙：只对匿名生效，所以单独存，不进入该用户的规则
			if after, isDeny := strings.CutPrefix(token, "!"); isDeny {
				rule, err := parseRuleToken(after)
				if err != nil {
					return err
				}
				if rule == nil {
					continue
				}
				rule.Perm = PermNO
				c.guestDenyRules = append(c.guestDenyRules, rule)
				continue
			}

			rule, err := parseRuleToken(token)
			if err != nil {
				return err
			}
			if rule == nil {
				continue
			}
			ac.Rules = append(ac.Rules, rule)
		}

		if len(ac.Rules) == 0 {
			continue
		}
		auths[ac.Username] = ac
	}

	// 匿名基线：只保留允许规则，供已登录用户回退使用（`!` 认证墙不算基线）
	if guest, ok := auths[""]; ok {
		for _, rule := range guest.Rules {
			if rule.Perm != PermNO {
				c.guestRules = append(c.guestRules, rule)
			}
		}
	}

	// 顶层 deny：对所有人硬拒绝
	for _, token := range c.Deny {
		rule, err := parseRuleToken(token)
		if err != nil {
			return err
		}
		if rule == nil {
			continue
		}
		c.hardDenyRules = append(c.hardDenyRules, rule)
	}

	c.parsedAuths = auths
	return nil
}

// CheckCredentials 验证用户名和密码是否匹配任意已配置用户
func (c *Config) CheckCredentials(username, password string) bool {
	auth, exists := c.parsedAuths[username]
	if !exists || auth.Username == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(auth.Password)) == 1
}

// AuthEnabled 是否启用认证
func (c *Config) AuthEnabled() bool {
	return len(c.parsedAuths) > 0
}

// IconCDNKeys 返回已配置的图标 CDN key（排序后，保证输出稳定）
func (c *Config) IconCDNKeys() []string {
	keys := make([]string, 0, len(c.IconsCDN))
	for k := range c.IconsCDN {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ParsedAuths 解析后的认证配置
func (c *Config) ParsedAuths() map[string]*AuthConfig {
	return c.parsedAuths
}

// SessionTTLDuration 返回会话有效期（从配置 server.session_ttl 解析），解析失败时返回默认 2h
func (c *Config) SessionTTLDuration() time.Duration {
	if c == nil {
		return 2 * time.Hour
	}

	raw := strings.TrimSpace(c.Server.SessionTTL)
	if raw == "" {
		return 2 * time.Hour
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		// 配置错误时回退默认值
		return 2 * time.Hour
	}
	return d
}

// normalizeCookieSecure 归一化 cookie_secure 取值，第二个返回值表示取值是否合法
func normalizeCookieSecure(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", cookieSecureAuto:
		return cookieSecureAuto, true
	case "true", "always", "force":
		return cookieSecureTrue, true
	case "false", "never", "off":
		return cookieSecureFalse, true
	}
	return "", false
}

// SecureCookie 判断当前请求下会话 cookie 是否应带 Secure 属性。
//
// 默认 auto：只有请求本身就是 HTTPS 时才带 Secure。本地明文 HTTP（例如
// http://localhost:8090）下若强制 Secure，浏览器会直接丢弃该 cookie，登录会失效。
// 部署在「TLS 终止的反向代理」之后时，容器里看到的仍是明文 HTTP，请显式配置
// cookie_secure: true。
func (c *Config) SecureCookie(r *http.Request) bool {
	if c == nil {
		return false
	}

	switch c.Server.CookieSecure {
	case cookieSecureTrue:
		return true
	case cookieSecureFalse:
		return false
	default: // auto
		return r != nil && r.TLS != nil
	}
}

// UserPermission 用户某路径的权限描述（面向前端展示）
type UserPermission struct {
	Path       string `json:"path"`
	Permission string `json:"perm"`
}

// UserPermissions 返回某个用户在配置中的权限规则列表，用于前端展示。
func (c *Config) UserPermissions(username string) []UserPermission {
	auth, exists := c.parsedAuths[username]
	if !exists {
		return nil
	}

	perms := make([]UserPermission, 0, len(auth.Rules))
	for _, rule := range auth.Rules {
		perms = append(perms, UserPermission{
			Path:       rule.displayPath(),
			Permission: string(rule.Perm),
		})
	}
	return perms
}
