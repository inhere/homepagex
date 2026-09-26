package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/inhere/homepagex/internal"
)

// 构建信息，由 Makefile / CI 通过 -ldflags -X main.xxx 注入
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var server *internal.Server

func main() {
	// 版本查询：./homepagex -V
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "-V", "--version", "version":
			fmt.Printf("homepagex %s (commit %s, built %s)\n", Version, GitCommit, BuildDate)
			return
		}
	}

	// 默认配置文件路径
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 加载配置
	config, err := internal.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: %v, using defaults", err)
	}

	// 初始化页面数据管理器
	internal.Init(config)
	server = internal.NewServer(config)
	mux := http.NewServeMux()

	// 注册路由
	registerRoutes(mux)

	// 启动服务器
	addr := fmt.Sprintf(":%s", config.Server.Port)
	log.Printf("🚀 Starting server on http://localhost%s", addr)
	log.Printf("Page data directory: %s", config.PagesDir)
	log.Printf("Frontend directory: %s", config.FrontendDir)
	fmt.Println()

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server listen failed: %v", err)
	}
}

func registerRoutes(mux *http.ServeMux) {
	// API 路由 - 统一的页面 API 处理器（支持 GET 和 POST）
	mux.HandleFunc("/api/page", server.BasicAuthMiddleware(server.PageApiHandler))
	mux.HandleFunc("/api/page/", server.BasicAuthMiddleware(server.PageApiHandler))

	// 登录/退出接口（UI 登录）
	mux.HandleFunc("/api/login", server.LoginHandler)
	mux.HandleFunc("/api/logout", server.LogoutHandler)

	// 图标缓存路由
	mux.HandleFunc("/icons-local/", server.GetIconLocalHandler)

	// 静态文件路由（前端应用）— 不需要认证，认证在 API 层处理
	mux.HandleFunc("/", server.StaticFileHandler)
}
