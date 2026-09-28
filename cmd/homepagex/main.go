package main

import (
	"flag"
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
	fs := flag.NewFlagSet("homepagex", flag.ExitOnError)

	var (
		showVersion bool
		configPath  string
		listenAddr  string
		modeFlag    string
	)

	fs.BoolVar(&showVersion, "V", false, "打印版本信息后退出")
	fs.BoolVar(&showVersion, "v", false, "print version and exit（同 -V）")
	fs.BoolVar(&showVersion, "version", false, "print version and exit（同 -V）")
	fs.StringVar(&configPath, "c", "config.yaml", "配置文件路径")
	fs.StringVar(&configPath, "config", "config.yaml", "config file path")
	fs.StringVar(&listenAddr, "addr", "", "监听地址，覆盖配置文件里的 server.port（如 :9090）")
	fs.StringVar(&modeFlag, "mode", "", "覆盖 server.mode：debug 或 release")

	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintf(out, "homepagex %s - 轻量级 Homer 风格导航主页\n\n", Version)
		fmt.Fprintf(out, "用法:\n  homepagex [选项] [配置文件]\n\n选项:\n")
		fs.PrintDefaults()
		fmt.Fprintf(out, "\n示例:\n")
		fmt.Fprintf(out, "  homepagex                    # 使用当前目录的 config.yaml\n")
		fmt.Fprintf(out, "  homepagex -c /etc/hpx.yaml   # 指定配置文件（也可直接跟位置参数）\n")
		fmt.Fprintf(out, "  homepagex --addr :9090       # 覆盖监听端口\n")
		fmt.Fprintf(out, "  homepagex -mode debug        # 以 debug 模式运行\n")
		fmt.Fprintf(out, "  homepagex -V                 # 查看版本\n")
	}

	// ExitOnError：参数写错时 flag 会打印错误 + 用法并退出 2
	fs.Parse(os.Args[1:])

	if showVersion {
		fmt.Printf("homepagex %s (commit %s, built %s)\n", Version, GitCommit, BuildDate)
		return
	}

	// 兼容旧用法：homepagex config.yaml
	if fs.NArg() > 0 {
		configPath = fs.Arg(0)
	}

	// 加载配置
	config, err := internal.LoadConfig(configPath)
	if err != nil {
		// config 非 nil 表示是「读不到文件」，有内置默认配置可兜底；
		// 为 nil 表示配置文件本身有问题（解析/校验失败），必须直接失败
		if config == nil {
			log.Fatalf("Invalid config file %q: %v", configPath, err)
		}
		log.Printf("Warning: %v, using built-in defaults", err)
	}

	// 命令行覆盖配置
	if modeFlag != "" {
		config.Server.Mode = modeFlag
	}
	if listenAddr == "" {
		listenAddr = ":" + config.Server.Port
	}

	// 初始化页面数据管理器
	internal.Init(config)
	server = internal.NewServer(config)
	mux := http.NewServeMux()

	// 注册路由
	registerRoutes(mux)

	// 启动服务器
	log.Printf("🚀 homepagex %s starting on http://localhost%s", Version, listenAddr)
	log.Printf("Config file: %s", configPath)
	log.Printf("Page data directory: %s", config.PagesDir)
	log.Printf("Frontend directory: %s", config.FrontendDir)
	fmt.Println()

	if err := http.ListenAndServe(listenAddr, mux); err != nil {
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
