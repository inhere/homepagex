package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
)

// newIconTestServer 构造一个带「假 CDN」的 Server，并返回 CDN 被请求的次数计数器。
// 计数器用来断言「到底有没有访问网络」，比看返回码更能说明问题。
func newIconTestServer(t *testing.T, iconsRemote bool, cdnHandler http.HandlerFunc) (*Server, *int64) {
	t.Helper()

	var count int64
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		cdnHandler(w, r)
	}))
	t.Cleanup(cdn.Close)

	cfg := &Config{
		Server:      ServerConfig{Mode: "release", Port: "0"},
		PagesDir:    t.TempDir(),
		FrontendDir: filepath.Join(t.TempDir(), "not-exist"),
		IconsDir:    t.TempDir(),
		IconsCDN:    map[string]string{"test-icons": cdn.URL + "/"},
		IconsRemote: iconsRemote,
	}
	return NewServer(cfg), &count
}

func doIconGet(srv *Server, iconPath string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.GetIconLocalHandler(rec, httptest.NewRequest(http.MethodGet, "/icons-local/"+iconPath, nil))
	return rec
}

// 离线部署：icons_remote=false 时缓存未命中直接 404，完全不碰网络
func TestIconRemoteDisabledNeverTouchesNetwork(t *testing.T) {
	srv, count := newIconTestServer(t, false, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("icons_remote=false 时不应发起任何下载请求: %s", r.URL)
		w.Write([]byte("should-not-be-called"))
	})

	rec := doIconGet(srv, "test-icons/png/plex.png")

	assert.Eq(t, http.StatusNotFound, rec.Code)
	assert.Eq(t, int64(0), atomic.LoadInt64(count))
}

// 缓存命中的图标直接读本地文件，同样不访问网络（离线时必须能出图）
func TestIconServedFromCacheWithoutNetwork(t *testing.T) {
	srv, count := newIconTestServer(t, true, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("缓存命中时不应发起下载请求: %s", r.URL)
	})

	cached := filepath.Join(srv.config.IconsDir, "test-icons", "png", "plex.png")
	assert.NoErr(t, os.MkdirAll(filepath.Dir(cached), 0o755))
	assert.NoErr(t, os.WriteFile(cached, []byte("cached-bytes"), 0o644))

	rec := doIconGet(srv, "test-icons/png/plex.png")

	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Eq(t, "cached-bytes", rec.Body.String())
	assert.True(t, rec.Header().Get("Content-Type") == "image/png")
	assert.Eq(t, int64(0), atomic.LoadInt64(count))
}

// 下载失败要落进失败缓存：静默期内不再重试（否则离线时每个请求都要等满 5s 超时）
func TestIconDownloadFailureIsCached(t *testing.T) {
	srv, count := newIconTestServer(t, true, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	t.Run("首次失败返回 302 兜底并记录失败", func(t *testing.T) {
		rec := doIconGet(srv, "test-icons/png/plex.png")

		assert.Eq(t, http.StatusFound, rec.Code)
		assert.Eq(t, int64(1), atomic.LoadInt64(count))
	})

	t.Run("静默期内不再重试", func(t *testing.T) {
		rec := doIconGet(srv, "test-icons/png/plex.png")

		assert.Eq(t, http.StatusFound, rec.Code)
		assert.Eq(t, int64(1), atomic.LoadInt64(count))
	})

	t.Run("静默期过后会重新尝试", func(t *testing.T) {
		srv.iconFailTTL = 20 * time.Millisecond
		time.Sleep(40 * time.Millisecond)

		rec := doIconGet(srv, "test-icons/png/plex.png")

		assert.Eq(t, http.StatusFound, rec.Code)
		assert.Eq(t, int64(2), atomic.LoadInt64(count))
	})
}

// 下载成功后写入 icons_dir，并且后续请求只读缓存
func TestIconDownloadWritesCacheDir(t *testing.T) {
	iconBytes := []byte("fake-png-bytes")
	srv, count := newIconTestServer(t, true, func(w http.ResponseWriter, r *http.Request) {
		assert.Eq(t, "/png/plex.png", r.URL.Path)
		w.Write(iconBytes)
	})

	rec := doIconGet(srv, "test-icons/png/plex.png")
	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Eq(t, string(iconBytes), rec.Body.String())
	assert.Eq(t, int64(1), atomic.LoadInt64(count))

	// 缓存目录可配置，文件必须落在 icons_dir 下（而不是内嵌 FS / frontend_dir）
	cached := filepath.Join(srv.config.IconsDir, "test-icons", "png", "plex.png")
	got, err := os.ReadFile(cached)
	assert.NoErr(t, err)
	assert.Eq(t, string(iconBytes), string(got))

	rec = doIconGet(srv, "test-icons/png/plex.png")
	assert.Eq(t, http.StatusOK, rec.Code)
	assert.Eq(t, int64(1), atomic.LoadInt64(count))
}

// 未配置的 CDN key 直接 404，不访问网络
func TestIconUnknownCDNKeyNotFound(t *testing.T) {
	srv, count := newIconTestServer(t, true, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("未知 CDN key 不应发起下载请求: %s", r.URL)
	})

	rec := doIconGet(srv, "unknown-icons/png/plex.png")

	assert.Eq(t, http.StatusNotFound, rec.Code)
	assert.Eq(t, int64(0), atomic.LoadInt64(count))
}
