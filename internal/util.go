package internal

import (
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
