package internal

import (
	"crypto/tls"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestParseAuths(t *testing.T) {
	tests := []struct {
		name    string
		auths   []string
		wantLen int
		checkFn func(t *testing.T, c *Config)
	}{
		{
			name:    "空配置",
			auths:   []string{},
			wantLen: 0,
		},
		{
			name:    "公开访问 @*",
			auths:   []string{"@*"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths[""]
				if auth == nil {
					t.Fatal("期望存在匿名配置")
				}
				assert.Eq(t, "", auth.Username)
				assert.Eq(t, "", auth.Password)
				assert.Eq(t, 1, len(auth.Rules))
				assert.Eq(t, PermRO, auth.Rules[0].Perm)
				assert.Eq(t, ruleAll, auth.Rules[0].Kind)
			},
		},
		{
			name:    "带用户名密码的完整配置",
			auths:   []string{"admin:admin123@*:rw"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths["admin"]
				if auth == nil {
					t.Fatal("期望找到用户 admin")
				}
				assert.Eq(t, "admin", auth.Username)
				assert.Eq(t, "admin123", auth.Password)
				assert.Eq(t, 1, len(auth.Rules))
				assert.Eq(t, PermRW, auth.Rules[0].Perm)
				assert.Eq(t, "/*:rw", auth.Rules[0].String())
			},
		},
		{
			name:    "多路径配置",
			auths:   []string{"user:pass@/api:rw,/static:ro"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths["user"]
				if auth == nil {
					t.Fatal("期望找到用户 user")
				}
				assert.Eq(t, "pass", auth.Password)
				assert.Eq(t, 2, len(auth.Rules))
				assert.Eq(t, "/api:rw", auth.Rules[0].String())
				assert.Eq(t, "/static:ro", auth.Rules[1].String())
			},
		},
		{
			name:    "匿名用户多路径",
			auths:   []string{"@/public:ro,/api"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths[""]
				if auth == nil {
					t.Fatal("期望存在匿名配置")
				}
				assert.Eq(t, "", auth.Username)
				assert.Eq(t, 2, len(auth.Rules))
				// 省略权限时默认 ro
				assert.Eq(t, "/api:ro", auth.Rules[1].String())
			},
		},
		{
			name:    "排除路径进入认证墙而不是用户规则",
			auths:   []string{"@*,!/inner"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths[""]
				if auth == nil {
					t.Fatal("期望存在匿名配置")
				}
				assert.Eq(t, 1, len(auth.Rules))
				assert.Eq(t, 1, len(c.guestDenyRules))
				assert.Eq(t, PermNO, c.guestDenyRules[0].Perm)
				// 匿名基线只含允许规则
				assert.Eq(t, 1, len(c.guestRules))
			},
		},
		{
			name:    "仅用户名无密码",
			auths:   []string{"admin@*:rw"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths["admin"]
				if auth == nil {
					t.Fatal("期望找到用户 admin")
				}
				assert.Eq(t, "admin", auth.Username)
				assert.Eq(t, "", auth.Password)
			},
		},
		{
			name:    "仅通配符路径",
			auths:   []string{"user:pass@*"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths["user"]
				if auth == nil {
					t.Fatal("期望找到用户 user")
				}
				assert.Eq(t, 1, len(auth.Rules))
				assert.Eq(t, PermRO, auth.Rules[0].Perm)
				assert.Eq(t, ruleAll, auth.Rules[0].Kind)
			},
		},
		{
			name:    "多用户配置",
			auths:   []string{"admin:admin123@*:rw", "user1:user123@/tools:rw"},
			wantLen: 2,
		},
		{
			name:    "空字符串过滤",
			auths:   []string{"", "admin:pass@*", ""},
			wantLen: 1,
		},
		{
			name:    "! 认证墙只对匿名生效",
			auths:   []string{"@*,!/inner*"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths[""]
				if auth == nil {
					t.Fatal("期望存在匿名配置")
				}
				assert.Eq(t, "", auth.Username)
				// 认证墙不进用户规则
				assert.Eq(t, 1, len(auth.Rules))
				assert.Eq(t, 1, len(c.guestDenyRules))
				// /inner* 归一化为前缀匹配 /inner
				assert.Eq(t, "/inner", c.guestDenyRules[0].Pattern)
				assert.Eq(t, rulePrefix, c.guestDenyRules[0].Kind)
			},
		},
		{
			name:    "显式 :no 不再被静默降级",
			auths:   []string{"user:pass@/secret:no,/tools:rw"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				auth := c.parsedAuths["user"]
				if auth == nil {
					t.Fatal("期望找到用户 user")
				}
				assert.Eq(t, PermNO, auth.Rules[0].Perm)
				assert.Eq(t, "/secret", auth.Rules[0].Pattern)
			},
		},
		{
			name:    "顶层 deny 解析为硬拒绝规则",
			auths:   []string{"admin:pass@*:rw"},
			wantLen: 1,
			checkFn: func(t *testing.T, c *Config) {
				assert.Eq(t, 0, len(c.hardDenyRules))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Auths: tt.auths}
			assert.NoErr(t, c.parseAuths())

			assert.Eq(t, tt.wantLen, len(c.parsedAuths))
			if tt.checkFn != nil {
				tt.checkFn(t, c)
			}
		})
	}
}

func TestParseDeny(t *testing.T) {
	c := &Config{
		Auths: []string{"admin:pass@*:rw"},
		Deny:  []string{"/inner-tools", "/secret/**"},
	}
	assert.NoErr(t, c.parseAuths())
	assert.Eq(t, 2, len(c.hardDenyRules))
	assert.Eq(t, "/inner-tools", c.hardDenyRules[0].Pattern)
	assert.Eq(t, "/secret", c.hardDenyRules[1].Pattern)
}

// 配置写错时必须启动失败，而不是静默降级成另一种语义
func TestParseAuthsError(t *testing.T) {
	t.Run("未知权限后缀", func(t *testing.T) {
		c := &Config{Auths: []string{"user:pass@/a:readonly"}}
		assert.Err(t, c.parseAuths())
	})

	t.Run("缺少 @ 分隔符", func(t *testing.T) {
		c := &Config{Auths: []string{"user:pass"}}
		assert.Err(t, c.parseAuths())
	})

	t.Run("deny 里的未知权限后缀", func(t *testing.T) {
		c := &Config{Auths: []string{"@*"}, Deny: []string{"/a:xx"}}
		assert.Err(t, c.parseAuths())
	})
}

func TestNormalizePattern(t *testing.T) {
	tests := []struct {
		raw      string
		wantPat  string
		wantKind ruleKind
	}{
		{"", "/", ruleAll},
		{"*", "/", ruleAll},
		{"/*", "/", ruleAll},
		{"/**", "/", ruleAll},
		{"/a", "/a", ruleSubtree},
		{"a", "/a", ruleSubtree},
		{"/a/**", "/a", ruleSubtree},
		{"/a/*", "/a", rulePrefix},
		{"/inner*", "/inner", rulePrefix},
		{"inner*", "/inner", rulePrefix},
		{"/a/b/c", "/a/b/c", ruleSubtree},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			pat, kind := normalizePattern(tt.raw)
			assert.Eq(t, tt.wantPat, pat)
			assert.Eq(t, tt.wantKind, kind)
		})
	}
}

func TestPathRuleMatch(t *testing.T) {
	tests := []struct {
		rule    string
		reqPath string
		want    bool
	}{
		// 前缀匹配（兼容旧写法）：/inner* 会命中 /inner-tools
		{"/inner*", "/inner", true},
		{"/inner*", "/inner-tools", true},
		{"/inner*", "/innerfoo", true},
		{"/inner*", "/other", false},
		// 子树匹配：/a 命中 /a 与 /a/x，但不命中 /ab
		{"/a", "/a", true},
		{"/a", "/a/x/y", true},
		{"/a", "/ab", false},
		// 全匹配
		{"*", "/anything", true},
		// 段内通配 /a/* 归一化为前缀 /a
		{"/a/*", "/a/x", true},
	}

	for _, tt := range tests {
		t.Run(tt.rule+" → "+tt.reqPath, func(t *testing.T) {
			rule, err := parseRuleToken(tt.rule)
			assert.NoErr(t, err)
			assert.Eq(t, tt.want, rule.match(tt.reqPath))
		})
	}
}

func TestIsNeedAuth(t *testing.T) {
	c := &Config{
		Auths: []string{"admin:admin123@*:rw", "user1:user123@/tools:rw", "@*,!/inner*"},
	}
	assert.NoErr(t, c.parseAuths())

	// 匿名：公开页面不需要登录
	assert.False(t, c.IsNeedAuth("/", false))
	// 匿名：认证墙内需要登录
	assert.True(t, c.IsNeedAuth("/inner-tools", false))
	// 匿名：写操作一律需要登录（即使基线是只读）
	assert.True(t, c.IsNeedAuth("/", true))
}

func TestConfigResolveRelativeDirs(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")

	t.Run("相对目录锚定到配置文件所在目录", func(t *testing.T) {
		cfg := &Config{PagesDir: "./pages", FrontendDir: "frontend/build", StaticDir: "./static", IconsDir: "./icons-cache"}
		assert.NoErr(t, cfg.ResolveRelativeDirs(configFile))

		assert.Eq(t, filepath.Join(dir, "pages"), cfg.PagesDir)
		assert.Eq(t, filepath.Join(dir, "frontend", "build"), cfg.FrontendDir)
		assert.Eq(t, filepath.Join(dir, "static"), cfg.StaticDir)
		assert.Eq(t, filepath.Join(dir, "icons-cache"), cfg.IconsDir)
	})

	t.Run("绝对目录保持不变", func(t *testing.T) {
		other := t.TempDir()
		cfg := &Config{PagesDir: other, FrontendDir: other, StaticDir: other, IconsDir: other}
		assert.NoErr(t, cfg.ResolveRelativeDirs(configFile))

		assert.Eq(t, other, cfg.PagesDir)
		assert.Eq(t, other, cfg.FrontendDir)
		assert.Eq(t, other, cfg.StaticDir)
		assert.Eq(t, other, cfg.IconsDir)
	})

	t.Run("空值保持为空", func(t *testing.T) {
		cfg := &Config{}
		assert.NoErr(t, cfg.ResolveRelativeDirs(configFile))

		assert.Eq(t, "", cfg.PagesDir)
		assert.Eq(t, "", cfg.FrontendDir)
		assert.Eq(t, "", cfg.StaticDir)
		assert.Eq(t, "", cfg.IconsDir)
	})
}

// 图标缓存目录：配置里没写时回落到 frontend_dir/icons-local（兼容直接构造 Config 的场景）
func TestConfigIconCacheDirFallback(t *testing.T) {
	cfg := &Config{FrontendDir: filepath.Join("root", "frontend", "build")}
	assert.Eq(t, filepath.Join("root", "frontend", "build", IconLocalPrefix), cfg.IconCacheDir())

	cfg.IconsDir = filepath.Join("root", "icons-cache")
	assert.Eq(t, filepath.Join("root", "icons-cache"), cfg.IconCacheDir())
}

// icons_dir / icons_remote：默认值 + 配置覆盖
func TestLoadConfigIconSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	t.Run("默认允许远程下载，缓存目录默认 icons-cache", func(t *testing.T) {
		assert.NoErr(t, os.WriteFile(path, []byte("server:\n  port: \"8090\"\n"), 0o644))

		cfg, err := LoadConfig(path)
		assert.NoErr(t, err)
		if err != nil {
			return
		}

		assert.True(t, cfg.IconsRemote)
		assert.Eq(t, "./icons-cache", cfg.IconsDir)

		assert.NoErr(t, cfg.ResolveRelativeDirs(path))
		assert.Eq(t, filepath.Join(dir, "icons-cache"), cfg.IconCacheDir())
	})

	t.Run("显式关闭远程下载并指定缓存目录", func(t *testing.T) {
		yamlText := "icons_remote: false\nicons_dir: \"cache/icons\"\n"
		assert.NoErr(t, os.WriteFile(path, []byte(yamlText), 0o644))

		cfg, err := LoadConfig(path)
		assert.NoErr(t, err)
		if err != nil {
			return
		}

		assert.False(t, cfg.IconsRemote)

		assert.NoErr(t, cfg.ResolveRelativeDirs(path))
		assert.Eq(t, filepath.Join(dir, "cache", "icons"), cfg.IconCacheDir())
	})
}

// 读不到配置文件时必须返回一份「可用的」默认配置而不是 nil，
// 否则调用方（main）会对着 nil 取字段，直接 panic
func TestLoadConfigMissingFileUsesDefaults(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "not-exist.yaml"))

	assert.Err(t, err)
	if cfg == nil {
		t.Fatal("读不到文件时应返回默认配置，而不是 nil")
	}

	assert.Eq(t, "8090", cfg.Server.Port)
	assert.Eq(t, "./pages", cfg.PagesDir)
	assert.Eq(t, "./frontend/build", cfg.FrontendDir)
	// 默认允许从 CDN 下载图标（离线部署可显式关掉）
	assert.True(t, cfg.IconsRemote)
	assert.Eq(t, "./icons-cache", cfg.IconsDir)

	// 默认配置是「匿名只读」，必须真的可用（auths 已解析）
	assert.True(t, cfg.Resolve("", "/", false).Allowed)
	assert.False(t, cfg.Resolve("", "/", true).Allowed)
}

// 文件存在但内容非法时必须返回 nil + err，让调用方直接失败，
// 而不是带着一份与用户预期不符的配置静默启动
func TestLoadConfigInvalidFileReturnsNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	assert.NoErr(t, os.WriteFile(path, []byte("auths:\n  - \"user:pass@/a:readonly\"\n"), 0o644))

	cfg, err := LoadConfig(path)

	assert.Err(t, err)
	assert.Nil(t, cfg)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Eq(t, "8090", cfg.Server.Port)
	assert.True(t, cfg.Resolve("", "/anything", false).Allowed)
	assert.False(t, cfg.Resolve("", "/anything", true).Allowed)
}

func TestNormalizeCookieSecure(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", cookieSecureAuto, true},
		{"auto", cookieSecureAuto, true},
		{"  AUTO ", cookieSecureAuto, true},
		{"true", cookieSecureTrue, true},
		{"always", cookieSecureTrue, true},
		{"false", cookieSecureFalse, true},
		{"never", cookieSecureFalse, true},
		{"yes", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := normalizeCookieSecure(tt.in)
			assert.Eq(t, tt.ok, ok)
			assert.Eq(t, tt.want, got)
		})
	}
}

func TestSecureCookie(t *testing.T) {
	httpsReq := httptest.NewRequest("GET", "/", nil)
	httpsReq.TLS = &tls.ConnectionState{}
	httpReq := httptest.NewRequest("GET", "/", nil)

	t.Run("auto（默认）只在 HTTPS 下带 Secure", func(t *testing.T) {
		cfg := &Config{}
		assert.False(t, cfg.SecureCookie(httpReq))
		assert.True(t, cfg.SecureCookie(httpsReq))
		assert.False(t, cfg.SecureCookie(nil))
	})

	t.Run("true 始终带 Secure", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{CookieSecure: cookieSecureTrue}}
		assert.True(t, cfg.SecureCookie(httpReq))
		assert.True(t, cfg.SecureCookie(httpsReq))
	})

	t.Run("false 始终不带 Secure", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{CookieSecure: cookieSecureFalse}}
		assert.False(t, cfg.SecureCookie(httpsReq))
	})
}

// cookie_secure 写错时必须启动失败，而不是静默回退
func TestLoadConfigRejectsInvalidCookieSecure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	assert.NoErr(t, os.WriteFile(path, []byte("server:\n  cookie_secure: maybe\n"), 0o644))

	cfg, err := LoadConfig(path)

	assert.Err(t, err)
	assert.Nil(t, cfg)
}
