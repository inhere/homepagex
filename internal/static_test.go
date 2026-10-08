package internal

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 外部目录无需包含骨架。覆盖内嵌和磁盘前端、同名资源保护及运行时更新。
func TestExternalStaticDirectory(t *testing.T) {
	for _, diskFrontend := range []bool{false, true} {
		t.Run(map[bool]string{false: "embedded", true: "disk"}[diskFrontend], func(t *testing.T) {
			root := t.TempDir()
			frontendDir := filepath.Join(root, "frontend")
			write := func(name, content string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if diskFrontend {
				write(filepath.Join(frontendDir, "index.html"), "<html>disk shell</html>")
				write(filepath.Join(frontendDir, "bundle.js"), "// disk bundle")
			}
			staticDir := filepath.Join(root, "static")
			srv := NewServer(&Config{FrontendDir: frontendDir, StaticDir: staticDir})
			// 可以在服务启动后创建目录和文件，不要求重新编译或重启。
			write(filepath.Join(staticDir, "custom.html"), "<html>external page</html>")
			write(filepath.Join(staticDir, "data", "status.json"), `{"version":1}`)
			write(filepath.Join(root, "outside.json"), "outside must not be served")
			for _, tc := range []struct{ path, body, contentType string }{
				{"/custom.html", "<html>external page</html>", "text/html"},
				{"/data/status.json", `{"version":1}`, "application/json"},
			} {
				rec := doStaticGet(srv, tc.path)
				if rec.Code != http.StatusOK || rec.Body.String() != tc.body || !strings.Contains(rec.Header().Get("Content-Type"), tc.contentType) {
					t.Fatalf("%s: status=%d, headers=%v, body=%s", tc.path, rec.Code, rec.Header(), rec.Body.String())
				}
			}
			// 外部资源不应覆盖首页及入口脚本，也不能破坏无扩展名的 YAML 路由。
			write(filepath.Join(staticDir, "index.html"), "external index must not override shell")
			write(filepath.Join(staticDir, "bundle.js"), "external bundle must not override shell")
			for _, name := range []string{"/", "/index.html", "/bundle.js", "/tools"} {
				rec := doStaticGet(srv, name)
				if rec.Code != http.StatusOK || rec.Body.Len() == 0 || strings.Contains(rec.Body.String(), "must not override") {
					t.Fatalf("shell resource %s overridden: %s", name, rec.Body.String())
				}
			}
			write(filepath.Join(staticDir, "data", "status.json"), `{"version":2}`)
			if got := doStaticGet(srv, "/data/status.json").Body.String(); got != `{"version":2}` {
				t.Fatalf("runtime edit not visible: %s", got)
			}
			for _, name := range []string{"/missing.json", "/../outside.json"} {
				if rec := doStaticGet(srv, name); rec.Code != http.StatusNotFound {
					t.Fatalf("%s should be 404, got %d", name, rec.Code)
				}
			}
		})
	}
}
