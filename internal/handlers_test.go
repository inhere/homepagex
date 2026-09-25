package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
