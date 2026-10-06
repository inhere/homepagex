package internal

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// APIResponse API 响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// defaultIconFailTTL 图标下载失败后的静默期：期间的请求不再重试下载。
//
// 离线/内网环境下 CDN 不可达，每个请求都要等满下载超时（5s）会让页面图标加载
// 极慢；记住失败结果就能把代价摊到每个图标每 TTL 一次。
const defaultIconFailTTL = 10 * time.Minute

// Server HTTP 服务器
type Server struct {
	config *Config
	// frontendFS 前端静态资源来源：frontend_dir 目录优先，其次内嵌资源（见 resolveFrontendFS）
	frontendFS http.FileSystem

	// 简单内存会话存储：sessionID -> Session
	sessions   map[string]*Session
	sessionsMu sync.RWMutex

	// 图标下载失败缓存：iconPath -> 失败时间。仅进程内有效，重启即清空。
	iconFailTTL time.Duration
	iconFails   map[string]time.Time
	iconFailsMu sync.Mutex
}

// Session 一个简单的会话对象
type Session struct {
	Username  string
	ExpiresAt time.Time
}

// NewServer 创建新的 HTTP 服务器
func NewServer(config *Config) *Server {
	return &Server{
		config:      config,
		frontendFS:  resolveFrontendFS(config.FrontendDir),
		sessions:    make(map[string]*Session),
		iconFailTTL: defaultIconFailTTL,
		iconFails:   make(map[string]time.Time),
	}
}

// sendJSON 发送 JSON 响应
func (s *Server) sendJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

// sendError 发送错误响应
func (s *Server) sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// debugf 只在 debug 模式输出日志，避免逐请求日志刷屏
func (s *Server) debugf(format string, args ...any) {
	if s.config == nil || s.config.Server.Mode != "debug" {
		return
	}
	log.Printf(format, args...)
}

// iconRecentlyFailed 该图标是否处于「下载失败静默期」内（见 defaultIconFailTTL）。
//
// 过期条目顺手删掉，避免缓存无限增长；返回值只表示「最近失败过，先别重试」。
func (s *Server) iconRecentlyFailed(iconPath string) bool {
	if s.iconFailTTL <= 0 {
		return false
	}

	s.iconFailsMu.Lock()
	defer s.iconFailsMu.Unlock()

	at, ok := s.iconFails[iconPath]
	if !ok {
		return false
	}
	if time.Since(at) >= s.iconFailTTL {
		delete(s.iconFails, iconPath)
		return false
	}
	return true
}

// markIconFailed 记录一次下载失败，TTL 内不再重试该图标
func (s *Server) markIconFailed(iconPath string) {
	if s.iconFailTTL <= 0 {
		return
	}

	s.iconFailsMu.Lock()
	defer s.iconFailsMu.Unlock()

	if s.iconFails == nil {
		s.iconFails = make(map[string]time.Time)
	}
	s.iconFails[iconPath] = time.Now()
}
