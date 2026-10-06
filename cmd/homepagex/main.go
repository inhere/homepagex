package main

import (
	"fmt"
	"net/http"

	"github.com/gookit/goutil/cflag/capp"
	"github.com/inhere/homepagex/internal"
)

// 构建信息。Version 是源码内置的兜底版本（本地 `go build` 时用），
// 发布构建由 Makefile / CI 通过 -ldflags -X main.Version=... 注入。
var (
	Version   = "0.3.0"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// globalOpts 全局选项
type globalOpts struct {
	// ConfigFile 配置文件路径，为空时按配置目录推导
	ConfigFile string
	// ConfigDir 配置目录，覆盖环境变量
	ConfigDir string
	// ShowVersion 打印版本后退出
	ShowVersion bool
}

var gOpts globalOpts

func main() {
	newApp().Run()
}

// newApp 组装命令行应用：全局选项 + 子命令
func newApp() *capp.App {
	app := capp.NewWith("homepagex", Version, "轻量级 Homer 风格导航主页")
	app.NameWidth = 8

	app.StringVar(&gOpts.ConfigFile, "config", "", "配置文件路径，默认 <配置目录>/config.yaml;false;c")
	app.StringVar(&gOpts.ConfigDir, "config-dir", "", "配置目录，默认 $HOMEPAGEX_CONFIG_DIR 或 ~/.config/homepagex")
	app.BoolVar(&gOpts.ShowVersion, "version", false, "打印版本信息后退出;false;V,v")

	// 全局选项解析完成、执行子命令之前：处理 --version
	app.OnAppFlagParsed = func(app *capp.App) bool {
		if gOpts.ShowVersion {
			printVersion()
			return false // 返回 false 停止后续流程
		}
		return true
	}

	app.Add(
		newServeCmd(),
		newInitCmd(),
		newFindCmd(),
		newOpenCmd(),
	)
	return app
}

func printVersion() {
	fmt.Printf("homepagex %s (commit %s, built %s)\n", Version, GitCommit, BuildDate)
}

// registerRoutes 注册所有 HTTP 路由
func registerRoutes(mux *http.ServeMux, server *internal.Server) {
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
