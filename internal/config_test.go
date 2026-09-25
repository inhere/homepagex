package internal

import (
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
