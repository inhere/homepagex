package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

const testHomeYAML = "title: \"Home\"\nservices:\n  - name: \"Media\"\n    items:\n      - name: \"Plex\"\n        url: \"https://plex.example.com\"\n"

// newTestServerFixture 准备一个 release 模式的测试服务与临时页面文件
func newTestServerFixture(t *testing.T, auths []string) (srv *Server, pagefile string, original string) {
	t.Helper()

	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	assert.NoErr(t, os.MkdirAll(pagesDir, 0o755))

	pagefile = filepath.Join(pagesDir, "home.yaml")
	original = testHomeYAML
	assert.NoErr(t, os.WriteFile(pagefile, []byte(original), 0o644))

	cfg := &Config{
		Server:      ServerConfig{Mode: "release", Port: "0", SessionTTL: "2h"},
		PagesDir:    pagesDir,
		FrontendDir: filepath.Join(root, "frontend"),
		Auths:       auths,
	}
	assert.NoErr(t, cfg.parseAuths())

	Init(cfg)
	return NewServer(cfg), pagefile, original
}

func newSaveRequest(t *testing.T, username, content string) *http.Request {
	t.Helper()

	body, err := json.Marshal(map[string]string{"content": content})
	assert.NoErr(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/page/?op=w", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if username != "" {
		req = req.WithContext(context.WithValue(req.Context(), ContextKeyUsername, username))
	}
	return req
}

// 覆盖 P0-1：YAML 校验失败必须返回非 2xx，否则前端会把失败当成功并丢弃用户修改
func TestHandlePagePostInvalidYAML(t *testing.T) {
	srv, pagefile, original := newTestServerFixture(t, []string{"admin:admin123@*:rw"})

	rec := httptest.NewRecorder()
	srv.PageApiHandler(rec, newSaveRequest(t, "admin", "title: \"Home\"\nservices: [\n"))

	assert.Eq(t, http.StatusBadRequest, rec.Code)

	var resp APIResponse
	assert.NoErr(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	assert.True(t, resp.Error != "")

	// 文件必须保持原样，不能被写坏
	got, err := os.ReadFile(pagefile)
	assert.NoErr(t, err)
	assert.Eq(t, original, string(got))
}

func TestHandlePagePostValidYAML(t *testing.T) {
	srv, pagefile, _ := newTestServerFixture(t, []string{"admin:admin123@*:rw"})

	newContent := "title: \"Home2\"\nservices: []\n"

	rec := httptest.NewRecorder()
	srv.PageApiHandler(rec, newSaveRequest(t, "admin", newContent))

	assert.Eq(t, http.StatusOK, rec.Code)

	var resp APIResponse
	assert.NoErr(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)

	got, err := os.ReadFile(pagefile)
	assert.NoErr(t, err)
	assert.Eq(t, newContent, string(got))

	// 保存后缓存必须失效，否则会继续返回旧配置
	page, err := PageDataMgr.GetPageConfig("/", false)
	assert.NoErr(t, err)
	assert.Eq(t, "Home2", page.Title)
}

func TestHandlePagePostRequiresLogin(t *testing.T) {
	srv, pagefile, original := newTestServerFixture(t, []string{"admin:admin123@*:rw"})

	rec := httptest.NewRecorder()
	srv.PageApiHandler(rec, newSaveRequest(t, "", "title: \"Hacked\"\n"))

	assert.Eq(t, http.StatusForbidden, rec.Code)

	got, err := os.ReadFile(pagefile)
	assert.NoErr(t, err)
	assert.Eq(t, original, string(got))
}

// 静态文件处理走的是 safeJoin，这里覆盖它的真实行为。
// 之所以专门加这个用例：safeJoin 曾在 Linux 上把 "/x.png" 判成绝对路径而拒绝
// （Windows 下 filepath.IsAbs 为 false，所以本地全过），结果是 Linux/Docker 里
// 静态文件与图标全部 404；只跑 Windows 的验证漏掉了，CI 的 ubuntu 矩阵才抓到。
func TestStaticFileHandler(t *testing.T) {
	srv, _, _ := newTestServerFixture(t, []string{"@*"})
	frontendDir := srv.config.FrontendDir

	assert.NoErr(t, os.MkdirAll(filepath.Join(frontendDir, "assets"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(frontendDir, "assets", "app.css"), []byte("body{}"), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(frontendDir, "index.html"), []byte("<html>spa</html>"), 0o644))
	assert.NoErr(t, os.MkdirAll(filepath.Join(frontendDir, "sub"), 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(frontendDir, "sub", "index.html"), []byte("<html>sub</html>"), 0o644))

	t.Run("带前导斜杠的静态文件可正常返回", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.StaticFileHandler(rec, httptest.NewRequest(http.MethodGet, "/assets/app.css", nil))

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Body.String(), "body{}"))
	})

	t.Run("目录请求返回目录下的 index.html", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.StaticFileHandler(rec, httptest.NewRequest(http.MethodGet, "/sub/", nil))

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Body.String(), "sub"))
	})

	t.Run("无扩展名的不存在路径交回前端路由", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.StaticFileHandler(rec, httptest.NewRequest(http.MethodGet, "/tools", nil))

		assert.Eq(t, http.StatusOK, rec.Code)
		assert.True(t, strings.Contains(rec.Body.String(), "spa"))
	})

	t.Run("不存在的资源返回 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.StaticFileHandler(rec, httptest.NewRequest(http.MethodGet, "/nope.png", nil))

		assert.Eq(t, http.StatusNotFound, rec.Code)
	})

	t.Run("目录穿越被拒绝", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.StaticFileHandler(rec, httptest.NewRequest(http.MethodGet, "/../config.yaml", nil))

		assert.Eq(t, http.StatusNotFound, rec.Code)
	})
}
