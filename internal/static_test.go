package internal

import (
	"net/http"
	"net/http/httptest"
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

func TestExternalStaticAuth(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	write := func(name, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(filepath.Join(staticDir, "ops", "sqlapi", "index.html"), "<html>ops</html>")
	write(filepath.Join(staticDir, "ops", "index.html"), "<html>ops root</html>")
	write(filepath.Join(staticDir, "ops", "edge-boards.json"), `{"ok":true}`)
	write(filepath.Join(staticDir, "ops-x", "index.html"), "<html>ops-x</html>")
	write(filepath.Join(staticDir, "public.json"), `{"public":true}`)

	cfg := &Config{
		StaticDir:   staticDir,
		FrontendDir: filepath.Join(root, "missing-frontend"),
		Auths:       []string{"@*,!/ops*", "admin:admin123@/ops*:ro", "blocked:blocked@/ops*:no"},
	}
	if err := cfg.parseAuths(); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(cfg)

	request := func(target, accept string, username string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		if username != "" {
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: srv.createSession(username)})
		}
		rec := httptest.NewRecorder()
		srv.StaticFileHandler(rec, req)
		return rec
	}

	t.Run("anonymous public external file remains public", func(t *testing.T) {
		rec := request("/public.json", "application/json", "")
		if rec.Code != http.StatusOK || rec.Body.String() != `{"public":true}` {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("anonymous protected JSON gets 401", func(t *testing.T) {
		rec := request("/ops/edge-boards.json", "application/json", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("anonymous protected HTML redirects to login with return path", func(t *testing.T) {
		rec := request("/ops/sqlapi/?tab=slow", "text/html", "")
		location := rec.Header().Get("Location")
		if rec.Code != http.StatusFound || !strings.HasPrefix(location, "/?login=1&redirect=") || !strings.Contains(location, "ops") {
			t.Fatalf("status=%d location=%q body=%q", rec.Code, location, rec.Body.String())
		}
	})

	t.Run("authorized user gets protected page", func(t *testing.T) {
		rec := request("/ops/sqlapi/", "text/html", "admin")
		if rec.Code != http.StatusOK || rec.Body.String() != "<html>ops</html>" {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("logged-in user without permission gets 403", func(t *testing.T) {
		rec := request("/ops/sqlapi/", "text/html", "blocked")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	for _, target := range []string{"/ops/../ops/edge-boards.json", "//ops//edge-boards.json", "/ops%2Fedge-boards.json", "/OPS/", "/ops-x"} {
		t.Run("path variant "+target, func(t *testing.T) {
			rec := request(target, "application/json", "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
			}
		})
	}

	t.Run("embedded frontend resource remains public", func(t *testing.T) {
		rec := request("/bundle.js", "application/javascript", "")
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Fatalf("status=%d body length=%d", rec.Code, rec.Body.Len())
		}
	})
}

func TestUppercaseAuthPatternProtectsStaticAndPagePaths(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(staticDir, "reports"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "reports", "r.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		StaticDir:   staticDir,
		FrontendDir: filepath.Join(root, "missing-frontend"),
		Auths:       []string{"@*,!/Reports*"},
	}
	if err := cfg.parseAuths(); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(cfg)

	t.Run("static file", func(t *testing.T) {
		rec := doStaticGet(srv, "/reports/r.json")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("page API", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/page/reports", nil)
		rec := httptest.NewRecorder()
		srv.BasicAuthMiddleware(srv.PageApiHandler)(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})
}
