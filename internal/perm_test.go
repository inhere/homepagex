package internal

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

// newPermConfig 构造一个用于权限测试的 Config
func newPermConfig(t *testing.T, auths []string, deny []string) *Config {
	t.Helper()

	c := &Config{Auths: auths, Deny: deny}
	assert.NoErr(t, c.parseAuths())
	return c
}

func TestResolveDefaultDeny(t *testing.T) {
	t.Run("没有任何规则命中", func(t *testing.T) {
		c := newPermConfig(t, nil, nil)
		acc := c.Resolve("", "/", false)
		assert.False(t, acc.Allowed)
		assert.Eq(t, PermNO, acc.Perm)
	})

	t.Run("配置里只有具名用户时匿名被拒绝", func(t *testing.T) {
		c := newPermConfig(t, []string{"admin:pass@*:rw"}, nil)
		assert.False(t, c.Resolve("", "/", false).Allowed)
		assert.True(t, c.Resolve("admin", "/", false).Allowed)
	})

	t.Run("匿名白名单之外被拒绝", func(t *testing.T) {
		c := newPermConfig(t, []string{"@/public:ro"}, nil)
		assert.True(t, c.Resolve("", "/public", false).Allowed)
		assert.False(t, c.Resolve("", "/private", false).Allowed)
	})
}

// 覆盖 P0-2：已登录用户在自己规则未命中时回退匿名基线，不能「登录后反而 403」
func TestResolveFallbackToGuestBaseline(t *testing.T) {
	c := newPermConfig(t, []string{
		"admin:admin123@*:rw",
		"user2:user123@/x:rw", // 故意不写 /*:ro 兜底
		"@*,!/inner*",
	}, nil)

	t.Run("用户规则未命中时沿用匿名基线", func(t *testing.T) {
		acc := c.Resolve("user2", "/", false)
		assert.True(t, acc.Allowed)
		assert.Eq(t, PermRO, acc.Perm)
		assert.False(t, acc.CanWrite)
	})

	t.Run("用户规则命中时优先使用自身规则", func(t *testing.T) {
		acc := c.Resolve("user2", "/x", false)
		assert.True(t, acc.Allowed)
		assert.Eq(t, PermRW, acc.Perm)
		assert.True(t, acc.CanWrite)
	})

	t.Run("匿名仍然被认证墙挡住", func(t *testing.T) {
		assert.False(t, c.Resolve("", "/inner-tools", false).Allowed)
	})

	t.Run("已登录后越过认证墙", func(t *testing.T) {
		acc := c.Resolve("user2", "/inner-tools", false)
		assert.True(t, acc.Allowed)
		assert.Eq(t, PermRO, acc.Perm)
	})
}

func TestResolveWriteRules(t *testing.T) {
	c := newPermConfig(t, []string{
		"admin:admin123@*:rw",
		"user1:user123@/tools:rw,/*:ro",
		"@*",
	}, nil)

	t.Run("写操作必须已登录（匿名 *:rw 也不行）", func(t *testing.T) {
		c2 := newPermConfig(t, []string{"@*:rw"}, nil)
		read := c2.Resolve("", "/", false)
		assert.True(t, read.Allowed)

		write := c2.Resolve("", "/", true)
		assert.False(t, write.Allowed)
		assert.False(t, write.CanWrite)
	})

	t.Run("只读路径不可写", func(t *testing.T) {
		acc := c.Resolve("user1", "/", true)
		assert.False(t, acc.Allowed)
		assert.False(t, acc.CanWrite)
	})

	t.Run("读写路径可写", func(t *testing.T) {
		acc := c.Resolve("user1", "/tools", true)
		assert.True(t, acc.Allowed)
		assert.True(t, acc.CanWrite)
	})

	t.Run("admin 全站可写", func(t *testing.T) {
		acc := c.Resolve("admin", "/anything/deep", true)
		assert.True(t, acc.Allowed)
		assert.True(t, acc.CanWrite)
	})
}

// 更具体的规则优先，而不是依赖配置数组顺序
func TestResolveSpecificity(t *testing.T) {
	c := newPermConfig(t, []string{"user:pass@/*:ro,/tools:rw"}, nil)

	assert.Eq(t, PermRW, c.Resolve("user", "/tools", false).Perm)
	assert.Eq(t, PermRW, c.Resolve("user", "/tools/sub", false).Perm)
	assert.Eq(t, PermRO, c.Resolve("user", "/other", false).Perm)
}

// 同等具体度冲突时取更严格的权限（no 优先），保证默认安全
func TestResolveSameSpecificityPrefersDeny(t *testing.T) {
	c := newPermConfig(t, []string{"user:pass@/*:no,/*:rw"}, nil)

	acc := c.Resolve("user", "/anything", false)
	assert.False(t, acc.Allowed)
	assert.Eq(t, PermNO, acc.Perm)
}

func TestResolveExplicitNoForUser(t *testing.T) {
	c := newPermConfig(t, []string{"user:pass@/inner-tools:no,/*:rw"}, nil)

	assert.False(t, c.Resolve("user", "/inner-tools", false).Allowed)
	assert.True(t, c.Resolve("user", "/other", true).Allowed)
}

// 顶层 deny 是无条件硬拒绝，连 admin 也不例外
func TestResolveHardDeny(t *testing.T) {
	c := newPermConfig(t,
		[]string{"admin:admin123@*:rw", "@*"},
		[]string{"/inner-tools"},
	)

	acc := c.Resolve("admin", "/inner-tools", false)
	assert.False(t, acc.Allowed)
	assert.Eq(t, PermNO, acc.Perm)

	// 未命中 deny 的照常
	assert.True(t, c.Resolve("admin", "/tools", false).Allowed)
	assert.True(t, c.Resolve("", "/tools", false).Allowed)
}

func TestFilterNavsByPermission(t *testing.T) {
	c := newPermConfig(t, []string{
		"admin:admin123@*:rw",
		"user1:user123@/tools:rw",
		"@*,!/inner*",
	}, nil)

	navs := []NavItem{
		{Name: "Home", URL: "/"},
		{Name: "Tools", URL: "/tools"},
		{Name: "InnerTools", URL: "/inner-tools"},
		{Name: "External", URL: "https://example.com"},
	}

	t.Run("匿名看不到认证墙内的导航", func(t *testing.T) {
		got := c.FilterNavsByPermission(navs, "")
		assert.Eq(t, 3, len(got))
		assert.Eq(t, "Home", got[0].Name)
		assert.Eq(t, "Tools", got[1].Name)
		assert.Eq(t, "External", got[2].Name)
	})

	t.Run("登录后可见认证墙内的导航", func(t *testing.T) {
		got := c.FilterNavsByPermission(navs, "user1")
		assert.Eq(t, 4, len(got))
	})

	t.Run("无权限的导航被过滤", func(t *testing.T) {
		c2 := newPermConfig(t, []string{"@/", "user:pass@/"}, nil)
		got := c2.FilterNavsByPermission(navs, "user")
		// 只保留 URL 为空或命中的项
		assert.Eq(t, 2, len(got))
	})
}
