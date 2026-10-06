package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gookit/goutil/cflag/capp"
	"github.com/gookit/goutil/cliutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/inhere/homepagex/frontend"
	"github.com/inhere/homepagex/internal"
)

// serveCmdOpts serve 命令选项
type serveCmdOpts struct {
	// Addr 监听地址，覆盖配置里的 server.port
	Addr string
	// Mode 运行模式，覆盖配置里的 server.mode
	Mode string
}

var serveOpts serveCmdOpts

func newServeCmd() *capp.Cmd {
	cmd := capp.NewCmd("serve", "启动导航主页服务", runServe)
	cmd.Aliases = []string{"s"}
	cmd.OnAdd = func(c *capp.Cmd) {
		c.StringVar(&serveOpts.Addr, "addr", "", "监听地址，覆盖配置里的 server.port（如 :9090）")
		c.StringVar(&serveOpts.Mode, "mode", "", "运行模式：debug 或 release，覆盖配置里的 server.mode")
	}
	return cmd
}

func runServe(c *capp.Cmd) error {
	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	if serveOpts.Mode != "" {
		config.Server.Mode = serveOpts.Mode
	}
	addr := serveOpts.Addr
	if addr == "" {
		addr = ":" + config.Server.Port
	}

	// 全局配置里 frontend_dir 通常指向配置目录，而发布包的前端跟在二进制旁边
	if dir := pickFrontendDir(config.FrontendDir); dir != config.FrontendDir {
		cliutil.Infoln("提示: 使用可执行文件旁的前端目录", dir)
		config.FrontendDir = dir
	}

	// 前端来源：目录里有 index.html 就用目录（便于开发与覆盖），否则用内嵌资源（单文件部署）
	frontendSource := "内嵌资源（embedded）"
	if hasIndexHTML(config.FrontendDir) {
		frontendSource = config.FrontendDir
	} else {
		cliutil.Infoln("提示: 前端目录", config.FrontendDir, "里没有 index.html，改用内嵌前端资源",
			"\n       源码运行请执行 `pnpm --dir frontend run build`，或修改配置里的 frontend_dir")
		if !frontend.Embedded {
			cliutil.Warnln("警告: 该二进制内嵌的是占位页（编译时未带 -tags embedfrontend）",
				"\n       请用 `make build` 重新编译，或让 frontend_dir 指向 pnpm 的构建产物")
		}
	}
	if !fsutil.IsDir(config.PagesDir) {
		cliutil.Warnln("警告: 页面目录不存在:", config.PagesDir, "\n       可运行 `homepagex init` 生成示例页面，或修改配置里的 pages_dir")
	}

	internal.Init(config)
	server := internal.NewServer(config)

	mux := http.NewServeMux()
	registerRoutes(mux, server)

	log.Printf("🚀 homepagex %s starting on http://localhost%s", Version, addr)
	log.Printf("Config file: %s", configPath)
	log.Printf("Page data directory: %s", config.PagesDir)
	log.Printf("Frontend source: %s", frontendSource)
	log.Printf("Icon cache directory: %s (icons_remote=%v)", config.IconCacheDir(), config.IconsRemote)
	log.Printf("Mode: %s", config.Server.Mode)

	return http.ListenAndServe(addr, mux)
}

// pickFrontendDir 解析前端目录：配置目录里没有 index.html 时，回退到可执行文件旁的 frontend/build。
//
// 这是「init -g + 发布包」的常见组合：配置在 ~/.config/homepagex，前端跟在二进制旁边。
func pickFrontendDir(configured string) string {
	if hasIndexHTML(configured) {
		return configured
	}

	exe, err := os.Executable()
	if err != nil {
		return configured
	}

	near := filepath.Join(filepath.Dir(exe), "frontend", "build")
	if hasIndexHTML(near) {
		return near
	}
	return configured
}

func hasIndexHTML(dir string) bool {
	return fsutil.IsFile(filepath.Join(dir, "index.html"))
}
