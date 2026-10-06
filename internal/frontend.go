package internal

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/inhere/homepagex/frontend"
)

// frontendIndex 前端入口文件名
const frontendIndex = "index.html"

// hasFrontendIndex 判断目录里是否有可直接使用的前端入口（index.html）
func hasFrontendIndex(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, frontendIndex))
	return err == nil && !info.IsDir()
}

// resolveFrontendFS 决定前端静态资源的来源。
//
// 优先级（与部署方式对应）：
//  1. 配置的 frontend_dir 里存在 index.html → 直接用该目录（开发时改完 pnpm build 立即生效，
//     也方便用别的目录覆盖内嵌资源）
//  2. 否则用内嵌的前端资源 → 单文件部署，二进制自带前端
//
// 注意：内嵌 FS 是只读的，前端产物里不能写任何运行时数据（图标缓存见 Config.IconCacheDir）。
func resolveFrontendFS(dir string) http.FileSystem {
	if hasFrontendIndex(dir) {
		return http.Dir(dir)
	}
	return http.FS(frontend.Assets())
}

// openFrontendFile 打开一个前端资源并归一化路径。
//
// name 是相对「站点根」的路径（请求路径或 "index.html"）：
//   - 路径统一用 path.Clean 清理，任何 ".." 都会被解析掉，不可能逃出资源根
//     （http.Dir / fs.FS 本身也拒绝非法路径，这里是纵深防御）
//   - 目录请求返回目录下的 index.html
//
// 返回打开的文件、归一化后的资源名、是否命中。
func (s *Server) openFrontendFile(name string) (http.File, string, bool) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || name == "." {
		name = frontendIndex
	}

	f, err := s.frontendFS.Open(name)
	if err != nil {
		return nil, name, false
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, name, false
	}

	if !info.IsDir() {
		return f, name, true
	}

	// 目录 → 目录下的 index.html
	f.Close()
	name = path.Join(name, frontendIndex)
	f, err = s.frontendFS.Open(name)
	if err != nil {
		return nil, name, false
	}
	if info, err = f.Stat(); err != nil || info.IsDir() {
		f.Close()
		return nil, name, false
	}
	return f, name, true
}
