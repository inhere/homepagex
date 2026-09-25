package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

// newAuthedRequest 构造一个携带指定用户 session 的请求
func newAuthedRequest(t *testing.T, srv *Server, username, method, target string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	if username != "" {
		sessionID := srv.createSession(username)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	}
	return req
}

func okHandler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}
}

// 覆盖 P0-2：登录用户对公开页面的访问不应被 403 拒绝
func TestBasicAuthMiddlewareLoggedInUserKeepsPublicAccess(t *testing.T) {
	cfg := &Config{
		Auths: []string{
			"user2:user123@/x:rw",
			"@*,!/inner*",
		},
	}
	assert.NoErr(t, cfg.parseAuths())
	srv := NewServer(cfg)
	mw := srv.BasicAuthMiddleware(okHandler(http.StatusOK))

	t.Run("匿名可读的首页，登录后仍可读", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mw(rec, newAuthedRequest(t, srv, "user2", http.MethodGet, "/api/page/"))
		assert.Eq(t, http.StatusOK, rec.Code)
	})

	t.Run("匿名可读的首页，匿名也可读", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mw(rec, newAuthedRequest(t, srv, "", http.MethodGet, "/api/page/"))
		assert.Eq(t, http.StatusOK, rec.Code)
	})

	t.Run("用户有 rw 的路径可写", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mw(rec, newAuthedRequest(t, srv, "user2", http.MethodPost, "/api/page/x?op=w"))
		assert.Eq(t, http.StatusOK, rec.Code)
	})

	t.Run("回退基线为 ro 的路径不可写", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mw(rec, newAuthedRequest(t, srv, "user2", http.MethodPost, "/api/page/?op=w"))
		assert.Eq(t, http.StatusForbidden, rec.Code)
	})

	t.Run("匿名写公开只读路径需要登录", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mw(rec, newAuthedRequest(t, srv, "", http.MethodPost, "/api/page/?op=w"))
		assert.Eq(t, http.StatusUnauthorized, rec.Code)
	})
}

// 匿名访问认证墙内的路径必须 401（前端据此弹登录框）
func TestBasicAuthMiddlewareGuestWall(t *testing.T) {
	cfg := &Config{Auths: []string{"@*,!/inner*"}}
	assert.NoErr(t, cfg.parseAuths())
	srv := NewServer(cfg)
	mw := srv.BasicAuthMiddleware(okHandler(http.StatusOK))

	rec := httptest.NewRecorder()
	mw(rec, newAuthedRequest(t, srv, "", http.MethodGet, "/api/page/inner-tools"))
	assert.Eq(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	mw(rec, newAuthedRequest(t, srv, "", http.MethodGet, "/api/page/tools"))
	assert.Eq(t, http.StatusOK, rec.Code)
}

// 没有任何规则的路径必须 fail closed
func TestBasicAuthMiddlewareFailClosed(t *testing.T) {
	cfg := &Config{Auths: []string{"admin:admin123@/admin:rw"}}
	assert.NoErr(t, cfg.parseAuths())
	srv := NewServer(cfg)
	mw := srv.BasicAuthMiddleware(okHandler(http.StatusOK))

	rec := httptest.NewRecorder()
	mw(rec, newAuthedRequest(t, srv, "", http.MethodGet, "/api/page/"))
	assert.Eq(t, http.StatusUnauthorized, rec.Code)

	// 已登录但自身规则未命中，且匿名基线为空 → 403
	rec = httptest.NewRecorder()
	mw(rec, newAuthedRequest(t, srv, "admin", http.MethodGet, "/api/page/"))
	assert.Eq(t, http.StatusForbidden, rec.Code)
}
