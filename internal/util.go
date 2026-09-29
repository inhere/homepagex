package internal

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// staticContentTypes 显式补齐常见静态资源类型。
//
// 之前只有 html/js/css/图片几种，导致 .woff2 字体和 .webp 图标都以
// application/octet-stream 返回（selfhst 的图标正是 webp）。
var staticContentTypes = map[string]string{
	".html":        "text/html; charset=utf-8",
	".js":          "text/javascript; charset=utf-8",
	".mjs":         "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".json":        "application/json; charset=utf-8",
	".map":         "application/json; charset=utf-8",
	".txt":         "text/plain; charset=utf-8",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".jpg":         "image/jpeg",
	".jpeg":        "image/jpeg",
	".gif":         "image/gif",
	".webp":        "image/webp",
	".avif":        "image/avif",
	".ico":         "image/x-icon",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".ttf":         "font/ttf",
	".otf":         "font/otf",
	".eot":         "application/vnd.ms-fontobject",
	".webmanifest": "application/manifest+json",
}

// getContentType 根据文件扩展名获取 Content-Type
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ct, ok := staticContentTypes[ext]; ok {
		return ct
	}
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// safeJoin 把一个「可能来自用户输入」的相对路径安全地拼接到 baseDir 之下。
//
// 这是全项目唯一做路径安全处理的地方：页面文件、静态文件、图标缓存都走它，
// 而不是各处各写一套 TrimLeft/TrimPrefix + filepath.Join。
//
// 净化用 Go 生态里公认的写法 `filepath.Clean("/" + p)`：前缀一个斜杠之后，
// 任何 ".." 都只会在根下被解析掉，结果必定以 "/" 开头，因此不可能逃出根；
// 再去掉这个前导斜杠，交给 filepath.Join 拼到 baseDir 之下。
// 这同时也是 CodeQL go/path-injection 认可的 sanitizer 写法。
//
// 另外做两件显式校验（纵深防御，也让行为可预期而不是静默改写）：
//   - 拒绝绝对路径与盘符（C:\\x、\\\\server\\share）
//   - 拒绝含 ".." 路径段的输入
//
// 返回值可直接用于 os.ReadFile / os.WriteFile / http.ServeFile。
func safeJoin(baseDir, relPath string) (string, error) {
	raw := strings.TrimSpace(relPath)
	if raw == "" {
		return "", errors.New("empty path")
	}

	// 统一分隔符：Windows 下反斜杠也是目录分隔符，先归一化再判断
	raw = strings.ReplaceAll(raw, "\\", "/")

	// 先去掉前导斜杠：调用方给的通常是 URL 路径（r.URL.Path 必定以 / 开头）或页面名
	// （如 /tools），它们是「相对站点根」而不是文件系统绝对路径。
	// 顺序很关键 —— 必须放在判断绝对路径之前：否则 Linux 下 filepath.IsAbs("/x.png")
	// 为 true，会被误判成绝对路径而拒绝；Windows 下同一条为 false。两平台行为不一致，
	// 表现为 Linux/Docker 里静态文件与图标全部 404（CI 的 ubuntu 矩阵就是这样抓到的）。
	raw = strings.TrimLeft(raw, "/")
	if raw == "" {
		return "", fmt.Errorf("empty path: %q", relPath)
	}

	// 去掉前导斜杠后仍是绝对路径的，只可能是带盘符/卷名的写法（如 C:/x）
	if filepath.IsAbs(raw) || filepath.VolumeName(raw) != "" {
		return "", fmt.Errorf("absolute path not allowed: %q", relPath)
	}
	for _, seg := range strings.Split(raw, "/") {
		if seg == ".." {
			return "", fmt.Errorf("path must not contain '..': %q", relPath)
		}
	}

	// 关键一步：Clean("/" + p) 保证结果以 "/" 开头，任何 ".." 都逃不出根
	clean := filepath.Clean("/" + raw)
	rel := strings.TrimPrefix(clean, "/")
	if rel == "" || rel == "." {
		return "", fmt.Errorf("empty path: %q", relPath)
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	target := filepath.Join(baseAbs, filepath.FromSlash(rel))

	// 纵深防御：再确认结果确实落在 baseDir 之内
	back, err := filepath.Rel(baseAbs, target)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes base dir: %q", relPath)
	}
	return target, nil
}

var client = &http.Client{
	Timeout: 5 * time.Second,
}

// downloadIconFile 下载图标文件到本地缓存
func downloadIconFile(remoteURL, localPath string) error {
	// 下载文件
	resp, err := client.Get(remoteURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download icon: %s, status: %d", remoteURL, resp.StatusCode)
	}

	dir := filepath.Dir(localPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 先写临时文件再 rename：避免下载中断留下半截文件被后续当成有效缓存
	tmp, err := os.CreateTemp(dir, ".icon-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err = io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, localPath)
}
