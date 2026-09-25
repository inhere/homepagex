package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type contextKey string

// ContextKeyUsername request context 中存储已认证用户名的 key
const ContextKeyUsername contextKey = "username"

const (
	sessionCookieName = "hpx_session"
)

func (s *Server) BasicAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.Path
		// 权限按页面配置，有页面的权限就有对应 api 的权限（api 访问去除 api/page 前缀后检查）
		if after, ok := strings.CutPrefix(reqPath, PageApiPrefix); ok {
			reqPath = after
		}

		isWrite := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions

		// 从 session cookie 中获取已登录用户
		username := s.usernameFromRequest(r)

		// 统一走 Resolve：用户规则优先、未命中回退匿名基线、命中 deny 一律拒绝
		acc := s.config.Resolve(username, reqPath, isWrite)
		if !acc.Allowed {
			if username == "" {
				// 未登录：401，由前端弹出登录 UI（不发送 WWW-Authenticate）
				s.sendError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			s.sendError(w, "Forbidden", http.StatusForbidden)
			return
		}

		// 将认证用户名注入 request context（游客为空字符串）
		ctx := context.WithValue(r.Context(), ContextKeyUsername, username)
		next(w, r.WithContext(ctx))
	}
}

// usernameFromRequest 从请求的 session cookie 中解析当前用户名
func (s *Server) usernameFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	s.sessionsMu.RLock()
	session, exists := s.sessions[cookie.Value]
	s.sessionsMu.RUnlock()

	if !exists {
		return ""
	}

	// 简单过期检查：过期则删除会话并视为未登录
	if time.Now().After(session.ExpiresAt) {
		s.deleteSession(cookie.Value)
		return ""
	}

	return session.Username
}

// createSession 创建新的会话并返回 sessionID
func (s *Server) createSession(username string) string {
	// 生成随机 sessionID
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// 理论上不会失败，失败时回退到时间戳+用户名
		log.Printf("createSession: rand.Read error: %v", err)
		return username + "-" + time.Now().Format(time.RFC3339Nano)
	}
	sessionID := hex.EncodeToString(buf)

	s.sessionsMu.Lock()
	s.sessions[sessionID] = &Session{
		Username:  username,
		ExpiresAt: time.Now().Add(s.config.SessionTTLDuration()),
	}
	s.sessionsMu.Unlock()

	return sessionID
}

// deleteSession 删除会话
func (s *Server) deleteSession(sessionID string) {
	if sessionID == "" {
		return
	}
	s.sessionsMu.Lock()
	delete(s.sessions, sessionID)
	s.sessionsMu.Unlock()
}

// LoginHandler UI 登录接口：验证用户名与密码，成功后设置 session cookie
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		s.sendError(w, "用户名和密码不能为空", http.StatusBadRequest)
		return
	}

	if !s.config.CheckCredentials(req.Username, req.Password) {
		// 统一错误信息，避免探测账号是否存在
		s.sendError(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	sessionID := s.createSession(req.Username)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// 不设置 Expires/MaxAge，浏览器会话结束即失效
	})

	// 返回当前用户信息和权限，便于前端更新状态与展示
	s.sendJSON(w, &LoginInfo{
		Username:    req.Username,
		Permissions: s.config.UserPermissions(req.Username),
	})
}

// LogoutHandler 退出登录：清理服务端会话并清除 cookie
func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.deleteSession(cookie.Value)
	}

	// 清除 cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	s.sendJSON(w, map[string]string{
		"message": "logged out",
	})
}
