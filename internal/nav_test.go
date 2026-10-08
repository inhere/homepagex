package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// 全局菜单与页面覆盖都必须把打开方式送到浏览器，且保留原有权限过滤。
func TestPageAPINavModes(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(map[bool]string{false: "global", true: "page"}[local], func(t *testing.T) {
			root := t.TempDir()
			navs := `
  - {name: Home, url: /}
  - {name: About, url: /about.html, mode: iframe}
  - {name: Standalone, url: /about.html, mode: page, target: _blank}
  - {name: Private, url: /private.html, mode: iframe}
`
			configYAML := "pages_dir: .\nauths: ['@*,!/private.html']\n"
			pageYAML := "title: Home\n"
			if local {
				pageYAML += "navs:" + navs
			} else {
				configYAML += "page_navs:" + navs
			}
			configPath := filepath.Join(root, "config.yaml")
			for name, data := range map[string]string{"config.yaml": configYAML, "home.yaml": pageYAML} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := LoadConfig(configPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.ResolveRelativeDirs(configPath); err != nil {
				t.Fatal(err)
			}
			Init(cfg)
			srv := NewServer(cfg)
			rec := httptest.NewRecorder()
			srv.BasicAuthMiddleware(srv.PageApiHandler)(rec, httptest.NewRequest(http.MethodGet, "/api/page", nil))
			var response struct {
				Success bool             `json:"success"`
				Data    PageDataResponse `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK || !response.Success || len(response.Data.Navs) != 3 {
				t.Fatalf("unexpected response: %s", rec.Body.String())
			}
			got := response.Data.Navs
			if got[0].Mode != "" || got[1].Mode != "iframe" || got[2].Mode != "page" || got[2].Target != "_blank" {
				t.Fatalf("nav modes lost: %+v", got)
			}
		})
	}
}
