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

// Server HTTP 服务器
type Server struct {
	config *Config
	fs     http.FileSystem

	// 简单内存会话存储：sessionID -> Session
	sessions   map[string]*Session
	sessionsMu sync.RWMutex
}

// Session 一个简单的会话对象
type Session struct {
	Username  string
	ExpiresAt time.Time
}

// NewServer 创建新的 HTTP 服务器
func NewServer(config *Config) *Server {
	return &Server{
		config:   config,
		fs:       http.Dir(config.FrontendDir),
		sessions: make(map[string]*Session),
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
