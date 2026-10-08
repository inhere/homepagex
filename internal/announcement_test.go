package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalAnnouncement(t *testing.T) {
	for _, tc := range []struct{ name, yaml, text string }{
		{"omitted", "", ""},
		{"empty", "announcement: ''\n", ""},
		{"whitespace", "announcement: '  '\n", "  "},
		{"multiline", "announcement: |\n  系统维护公告\n\n  <strong>保留为纯文本</strong>\n", "系统维护公告\n\n<strong>保留为纯文本</strong>\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pages := newTestPageManager(t, map[string]string{"home": "title: Home\n", "tools": "title: Tools\n"})
			configPath := filepath.Join(pages.PageDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte("pages_dir: .\n"+tc.yaml), 0o644); err != nil {
				t.Fatal(err)
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
			for _, route := range []string{"/api/page", "/api/page/tools"} {
				rec := httptest.NewRecorder()
				srv.BasicAuthMiddleware(srv.PageApiHandler)(rec, httptest.NewRequest(http.MethodGet, route, nil))
				var response struct {
					Success bool             `json:"success"`
					Data    PageDataResponse `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if rec.Code != http.StatusOK || !response.Success || response.Data.Announcement != tc.text {
					t.Fatalf("%s: expected announcement %q, response=%s", route, tc.text, rec.Body.String())
				}
			}
		})
	}
}
