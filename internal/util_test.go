package internal

import (
	"path/filepath"
	"runtime"
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

	t.Run("盘符路径被拒绝（仅 Windows 有卷名概念）", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Linux 下 C:/x 只是普通相对名，由归属校验兜底")
		}
		_, err := safeJoin(base, `C:\Windows\win.ini`)
		assert.Err(t, err)
	})

	t.Run("拒绝空路径", func(t *testing.T) {
		_, err := safeJoin(base, "   ")
		assert.Err(t, err)
	})

	// 这才是 safeJoin 真正要保证的东西（也是 go/path-injection 要拦的）：
	// 不管输入长什么样，要么直接报错，要么结果必须落在 baseDir 之内。
	// 两个平台对「绝对路径」的判定不同（Linux 认为 /x 是绝对路径、Windows 不认为），
	// 所以这里断言的是不变量，而不是某个平台的判定细节。
	t.Run("要么报错，要么结果落在 baseDir 之内", func(t *testing.T) {
		inputs := []string{
			"a/b/c.yaml", "/x.png", "deep/deeper/f.txt",
			"/", "   ", "../secret.yaml", "a/../../secret.yaml",
			`a\..\..\secret.yaml`, "C:/Windows/win.ini", "//server/share/x",
			"/a/b/../../../etc/passwd",
		}

		for _, rel := range inputs {
			got, err := safeJoin(base, rel)
			if err != nil {
				continue // 拒绝也是正确处理
			}

			back, rerr := filepath.Rel(base, got)
			assert.NoErr(t, rerr)
			assert.False(t,
				back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)),
				"输入 %q 的结果逃出了 baseDir: %s", rel, got)
		}
	})
}
