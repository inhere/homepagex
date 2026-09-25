package internal

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/gookit/goutil/strutil"
)

const (
	// IconLocalPrefix 图标缓存路径前缀
	IconLocalPrefix = "icons-local"
	PageApiPrefix   = "/api/page"
)

// HealthHandler 健康检查
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, map[string]string{"status": "ok"})
}

// GetIconLocalHandler 图标缓存处理
// 当 icon 路径以 icons-local/ 开头时，从本地缓存读取，若不存在则下载并缓存
func (s *Server) GetIconLocalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取图标路径 (去掉 icons-local/ 前缀)
	iconPath := strings.TrimPrefix(r.URL.Path, "/icons-local/")
	if iconPath == "" {
		s.sendError(w, "Icon path required", http.StatusBadRequest)
		return
	}

	// 构建本地缓存路径
	cacheDir := filepath.Join(s.config.FrontendDir, IconLocalPrefix)
	localPath := filepath.Join(cacheDir, iconPath)

	// 检查本地缓存是否存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		// 本地不存在，尝试下载
		iconCdnKey := strutil.BeforeFirst(iconPath, "/")
		baseRemoteUrl, ok := s.config.IconsCDN[iconCdnKey]
		if !ok {
			log.Printf("Icon CDN key %q not found in config.icons_cdn", iconCdnKey)
			s.sendError(w, "Icon not found", http.StatusNotFound)
			return
		}

		// IconsCDN[iconCdnKey] 是 CDN 的基础前缀，例如：
		//   dashboard-icons: https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/
		//   selfhst-icons:   https://cdn.jsdelivr.net/gh/selfhst/icons/
		// 本地访问路径形如：{cdn-key}/webp/openobserve.webp
		// 所以拼远程 URL 时需要去掉本地路径中的 {cdn-key}/ 前缀。
		relPath := strings.TrimPrefix(iconPath, iconCdnKey+"/")
		remoteURL := baseRemoteUrl + relPath

		log.Printf("Icon cache miss: %s, downloading from: %s", iconPath, remoteURL)

		// 下载文件
		if err := downloadIconFile(remoteURL, localPath); err != nil {
			// 下载失败（CDN 抖动、图标名不对等）时不要让整张图 500：
			// 退回 302 让浏览器直连 CDN 自行兜底。
			log.Printf("WARN failed to download icon: %v", err)
			http.Redirect(w, r, remoteURL, http.StatusFound)
			return
		}

		log.Printf("Icon cached: %s -> %s", iconPath, localPath)
	}

	// 读取并返回本地文件
	data, err := os.ReadFile(localPath)
	if err != nil {
		log.Printf("Failed to read cached icon: %v", err)
		s.sendError(w, "Icon read error", http.StatusInternalServerError)
		return
	}

	writeIconBytes(w, localPath, data)
}

// writeIconBytes 输出图标字节并设置缓存头
func writeIconBytes(w http.ResponseWriter, path string, data []byte) {
	w.Header().Set("Content-Type", getContentType(path))
	w.Header().Set("Cache-Control", "public, max-age=86400") // 缓存 24 小时
	w.Write(data)
}

// PageApiHandler 页面 API 统一处理器
// GET  /api/page/xxx             - 获取页面配置
// GET  /api/page/xxx?op=r        - 获取原始 YAML 内容
// GET  /api/page/xxx?op=blocks   - 获取可单独编辑的块（分组 / 条目）
// POST /api/page/xxx?op=w        - 保存整份 YAML 内容
// POST /api/page/xxx?op=block    - 修改单个块（update / insert / delete）
func (s *Server) PageApiHandler(w http.ResponseWriter, r *http.Request) {
	// 从 URL 路径获取路由
	path := strings.TrimPrefix(r.URL.Path, "/api/page")
	if path == "" {
		path = "/"
	}

	op := r.URL.Query().Get("op")

	// 根据请求方法分发
	switch r.Method {
	case http.MethodGet:
		switch op {
		case "r":
			s.getPageRawContent(w, r, path)
		case "blocks":
			s.getPageBlocks(w, r, path)
		default:
			s.handlePageGet(w, r, path)
		}
	case http.MethodPost:
		switch op {
		case "block":
			s.postPageBlock(w, r, path)
		default:
			s.handlePagePost(w, r, path)
		}
	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePageGet 处理 GET 请求
func (s *Server) handlePageGet(w http.ResponseWriter, r *http.Request, path string) {
	// 检查 refresh 参数
	refresh := strutil.SafeBool(r.URL.Query().Get("refresh"))

	pageConfig, err := PageDataMgr.GetPageConfig(path, refresh)
	if err != nil {
		log.Printf("Error loading page data for %s: %v", r.URL.Path, err)
		s.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	// 获取当前认证用户（由 BasicAuthMiddleware 注入 context）
	username, _ := r.Context().Value(ContextKeyUsername).(string)

	// 当前身份对该页面的访问权限：can_write 下发给前端，避免前端重写一份匹配逻辑
	acc := s.config.Resolve(username, path, true)

	// 构建响应：过滤导航项 + 附加用户信息
	resp := &PageDataResponse{
		PageConfig:  pageConfig,
		Navs:        s.config.FilterNavsByPermission(pageConfig.Navs, username),
		CanWrite:    acc.CanWrite,
		IconCDNKeys: s.config.IconCDNKeys(),
	}
	if username != "" {
		resp.UserInfo = &LoginInfo{
			Username:    username,
			Permissions: s.config.UserPermissions(username),
		}
	}

	log.Printf("Request API GET %s, Pagefile: %s, User: %q", r.RequestURI, pageConfig.Pagefile, username)
	s.sendJSON(w, resp)
}

// handlePagePost 处理 POST 请求（保存 YAML）
func (s *Server) handlePagePost(w http.ResponseWriter, r *http.Request, path string) {
	// 获取当前认证用户
	username, _ := r.Context().Value(ContextKeyUsername).(string)

	// 写操作必须已登录且对该页面有 rw 权限。
	// 中间件已经判过，这里做纵深防御（也覆盖直接调用 handler 的场景）。
	if !s.config.Resolve(username, path, true).CanWrite {
		s.sendError(w, "需要登录并具备读写权限后才能修改配置", http.StatusForbidden)
		return
	}

	// 解析请求体（只需要 content）
	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取页面配置以获取文件路径
	pageConfig, err := PageDataMgr.GetPageConfig(path, false)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	// 验证 YAML 格式
	var testConfig PageConfig
	if err := yaml.Unmarshal([]byte(req.Content), &testConfig); err != nil {
		// 必须返回非 2xx：否则前端会把校验失败当成保存成功，导致用户的修改被静默丢弃
		s.sendError(w, "YAML 格式错误: "+yaml.FormatError(err, false, true), http.StatusBadRequest)
		return
	}

	// 写入文件：先备份，再原子写入（避免写一半损坏配置）
	if err := backupFile(pageConfig.Pagefile); err != nil {
		log.Printf("Error backing up file %s: %v", pageConfig.Pagefile, err)
		s.sendError(w, "Failed to backup file", http.StatusInternalServerError)
		return
	}
	if err := writeFileAtomic(pageConfig.Pagefile, []byte(req.Content), 0644); err != nil {
		log.Printf("Error writing file %s: %v", pageConfig.Pagefile, err)
		s.sendError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// 清除缓存 - 强制下次重新加载
	PageDataMgr.ClearCache(path)

	log.Printf("Page config saved: %s by user: %s", pageConfig.Pagefile, username)

	s.sendJSON(w, map[string]interface{}{
		"success": true,
		"message": "保存成功",
	})
}

// getPageBlocks 返回页面里所有可单独编辑的块
func (s *Server) getPageBlocks(w http.ResponseWriter, r *http.Request, path string) {
	pageConfig, err := PageDataMgr.GetPageConfig(path, false)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	content, err := os.ReadFile(pageConfig.Pagefile)
	if err != nil {
		log.Printf("Error reading file %s: %v", pageConfig.Pagefile, err)
		s.sendError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	blocks, err := listPageBlocks(content)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	s.sendJSON(w, map[string]interface{}{
		"blocks": blocks,
	})
}

// postPageBlock 修改单个块（update / insert / delete），其余部分保持原样
func (s *Server) postPageBlock(w http.ResponseWriter, r *http.Request, path string) {
	username, _ := r.Context().Value(ContextKeyUsername).(string)

	if !s.config.Resolve(username, path, true).CanWrite {
		s.sendError(w, "需要登录并具备读写权限后才能修改配置", http.StatusForbidden)
		return
	}

	var req BlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pageConfig, err := PageDataMgr.GetPageConfig(path, false)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	content, err := os.ReadFile(pageConfig.Pagefile)
	if err != nil {
		log.Printf("Error reading file %s: %v", pageConfig.Pagefile, err)
		s.sendError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	updated, err := applyPageBlock(content, req)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 纵深防御：改动后的整份配置必须仍然可解析
	var testConfig PageConfig
	if err := yaml.Unmarshal(updated, &testConfig); err != nil {
		s.sendError(w, "改动后的配置无法解析: "+yaml.FormatError(err, false, true), http.StatusBadRequest)
		return
	}

	if err := backupFile(pageConfig.Pagefile); err != nil {
		log.Printf("Error backing up file %s: %v", pageConfig.Pagefile, err)
		s.sendError(w, "Failed to backup file", http.StatusInternalServerError)
		return
	}
	if err := writeFileAtomic(pageConfig.Pagefile, updated, 0644); err != nil {
		log.Printf("Error writing file %s: %v", pageConfig.Pagefile, err)
		s.sendError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	PageDataMgr.ClearCache(path)

	log.Printf("Page block %s saved: %s by user: %s", req.Action, pageConfig.Pagefile, username)

	s.sendJSON(w, map[string]interface{}{
		"success": true,
		"message": "保存成功",
	})
}

// StaticFileHandler 静态文件服务
func (s *Server) StaticFileHandler(w http.ResponseWriter, r *http.Request) {
	// 清理路径防止目录遍历
	path := strings.TrimLeft(r.URL.Path, "/.")
	if path == "" {
		path = "index.html"
	}

	// 构建完整路径
	fullPath := filepath.Join(s.config.FrontendDir, path)

	// 检查文件是否存在
	info, err := os.Stat(fullPath)
	if err != nil {
		// 如果是目录，尝试 index.html
		if info != nil && info.IsDir() {
			fullPath = filepath.Join(fullPath, "index.html")
		} else {
			extName := filepath.Ext(path)
			if extName == "" {
				// 返回前端应用的 index.html（支持前端路由）
				fullPath = filepath.Join(s.config.FrontendDir, "index.html")
			} else {
				// 逐请求的静态资源日志太吵，只在 debug 模式打印
				s.debugf("NOTICE File not found: %s", fullPath)
				// 否则返回 404
				s.sendError(w, "File not found", http.StatusNotFound)
				return
			}
		}
	}

	s.debugf("Request static: %s, Serving file: %s", r.URL.Path, fullPath)

	// 设置正确的 Content-Type
	contentType := getContentType(fullPath)
	w.Header().Set("Content-Type", contentType)

	http.ServeFile(w, r, fullPath)
}

// getPageRawContent 获取页面原始 YAML 内容（内部方法）
func (s *Server) getPageRawContent(w http.ResponseWriter, r *http.Request, path string) {
	// 获取页面配置以获取文件路径
	pageConfig, err := PageDataMgr.GetPageConfig(path, false)
	if err != nil {
		log.Printf("Error loading page config for %s: %v", path, err)
		s.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	// 读取原始 YAML 文件内容
	content, err := os.ReadFile(pageConfig.Pagefile)
	if err != nil {
		log.Printf("Error reading file %s: %v", pageConfig.Pagefile, err)
		s.sendError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	log.Printf("Request API GET %s, returning raw YAML", r.RequestURI)
	// 只返回文件内容：不要把服务器上的绝对路径暴露给浏览器
	s.sendJSON(w, map[string]string{
		"content": string(content),
	})
}
