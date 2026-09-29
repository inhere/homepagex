package internal

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestGetContentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		// 字体：之前会以 application/octet-stream 返回
		{"ajax/libs/font-awesome/6.4.0/webfonts/fa-solid-900.woff2", "font/woff2"},
		{"font.woff", "font/woff"},
		{"font.ttf", "font/ttf"},
		// selfhst 的图标就是 webp
		{"icons-local/selfhst-icons/webp/openobserve.webp", "image/webp"},
		{"bundle.js", "text/javascript; charset=utf-8"},
		{"bundle.css", "text/css; charset=utf-8"},
		{"index.html", "text/html; charset=utf-8"},
		{"bundle.js.map", "application/json; charset=utf-8"},
		{"logo.svg", "image/svg+xml"},
		{"logo.png", "image/png"},
		{"favicon.ico", "image/x-icon"},
		{"unknown.bin", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Eq(t, tt.want, getContentType(tt.path))
		})
	}
}

// safeJoin 是全项目唯一的路径安全入口：清理 + 归属校验
func TestSafeJoin(t *testing.T) {
	base := filepath.Join(t.TempDir(), "pages")

	// 相对 baseDir 的结果路径，便于断言
	joined := func(rel string) string {
		got, err := safeJoin(base, rel)
		assert.NoErr(t, err)
		back, rerr := filepath.Rel(base, got)
		assert.NoErr(t, rerr)
		return back
	}

	t.Run("普通相对路径", func(t *testing.T) {
		assert.Eq(t, "home.yaml", joined("home.yaml"))
	})

	t.Run("多级相对路径", func(t *testing.T) {
		assert.Eq(t, filepath.Join("sub", "home.yaml"), joined("sub/home.yaml"))
	})

	t.Run("URL 形式的前导斜杠会被去掉", func(t *testing.T) {
		assert.Eq(t, filepath.Join("ajax", "libs", "all.min.css"), joined("/ajax/libs/all.min.css"))
	})

	t.Run("反斜杠作为分隔符也被归一化", func(t *testing.T) {
		assert.Eq(t, filepath.Join("sub", "home.yaml"), joined(`sub\home.yaml`))
	})

	t.Run("拒绝 .. 段", func(t *testing.T) {
		_, err := safeJoin(base, "../secret.yaml")
		assert.Err(t, err)
	})

	t.Run("拒绝内嵌 .. 段", func(t *testing.T) {
		_, err := safeJoin(base, "a/../../secret.yaml")
		assert.Err(t, err)
	})

	t.Run("拒绝反斜杠形式的 .. 段", func(t *testing.T) {
		_, err := safeJoin(base, `a\..\..\secret.yaml`)
		assert.Err(t, err)
	})

	t.Run("拒绝绝对路径", func(t *testing.T) {
		_, err := safeJoin(base, filepath.Join(base, "home.yaml"))
		assert.Err(t, err)
	})

	t.Run("拒绝空路径", func(t *testing.T) {
		_, err := safeJoin(base, "   ")
		assert.Err(t, err)
	})

	t.Run("结果一定落在 baseDir 之内", func(t *testing.T) {
		for _, rel := range []string{"a/b/c.yaml", "/x.png", "deep/deeper/f.txt"} {
			got, err := safeJoin(base, rel)
			assert.NoErr(t, err)
			assert.True(t, strings.HasPrefix(got, base))
		}
	})
}
