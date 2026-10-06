package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

// newStaticTestServer 构造一个只关心静态资源来源的 Server。
//
// frontendDir 不存在（或没有 index.html）时，前端资源应当来自内嵌 FS。
func newStaticTestServer(t *testing.T, frontendDir string) *Server {
	t.Helper()

	cfg := &Config{
		Server:      ServerConfig{Mode: "release", Port: "0"},
		PagesDir:    t.TempDir(),
		FrontendDir: frontendDir,
		IconsDir:    t.TempDir(),
	}
	return NewServer(cfg)
}

func doStaticGet(srv *Server, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.StaticFileHandler(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// 单文件部署：配置文件里的 frontend_dir 不存在时，前端必须由内嵌资源提供
func TestStaticFileHandlerUsesEmbeddedWhenNoFrontendDir(t *testing.T) {
	srv := newStaticTestServer(t, filepath.Join(t.TempDir(), "not-exist"))

	t.Run("GET / 返回内嵌 index.html", func(t *testing.T) {
		rec := doStaticGet(srv, "/")

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Header().Get("Content-Type"), "text/html"))
		assert.True(t, rec.Body.Len() > 0)
	})

	// 前端入口脚本也必须来自内嵌资源，否则单文件部署的页面会白屏
	t.Run("GET /bundle.js 返回内嵌脚本", func(t *testing.T) {
		rec := doStaticGet(srv, "/bundle.js")

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Header().Get("Content-Type"), "javascript"))
		assert.True(t, rec.Body.Len() > 0)
	})

	t.Run("前端路由路径回落到内嵌 index.html", func(t *testing.T) {
		rec := doStaticGet(srv, "/tools")

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Header().Get("Content-Type"), "text/html"))
	})

	t.Run("不存在的静态资源返回 404", func(t *testing.T) {
		rec := doStaticGet(srv, "/nope.png")

		assert.Eq(t, http.StatusNotFound, rec.Code)
	})
}

// frontend_dir 里有 index.html 时优先用目录（开发时改完 pnpm build 立即生效，也能覆盖内嵌资源）
func TestStaticFileHandlerPrefersFrontendDirOverEmbedded(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	assert.NoErr(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>from-dir</html>"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "bundle.js"), []byte("// from-dir"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "assets", "app.css"), []byte("body{color:red}"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "sub", "index.html"), []byte("<html>sub</html>"), 0o644))

	srv := newStaticTestServer(t, dir)

	t.Run("首页来自目录", func(t *testing.T) {
		rec := doStaticGet(srv, "/")

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Body.String(), "from-dir"))
	})

	t.Run("入口脚本来自目录", func(t *testing.T) {
		rec := doStaticGet(srv, "/bundle.js")

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Body.String(), "from-dir"))
	})

	t.Run("子目录资源带正确 Content-Type", func(t *testing.T) {
		rec := doStaticGet(srv, "/assets/app.css")

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Header().Get("Content-Type"), "text/css"))
		assert.True(t, strings.Contains(rec.Body.String(), "body{color:red}"))
	})

	t.Run("目录请求返回目录下的 index.html", func(t *testing.T) {
		rec := doStaticGet(srv, "/sub/")

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Body.String(), "sub"))
	})
}
