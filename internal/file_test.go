package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/gookit/goutil/testutil/assert"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.yaml")

	assert.NoErr(t, writeFileAtomic(path, []byte("title: A\n"), 0o644))
	got, err := os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, "title: A\n", string(got))

	// 覆盖写入
	assert.NoErr(t, writeFileAtomic(path, []byte("title: B\n"), 0o644))
	got, err = os.ReadFile(path)
	assert.NoErr(t, err)
	assert.Eq(t, "title: B\n", string(got))

	// 不应留下临时文件
	entries, err := os.ReadDir(dir)
	assert.NoErr(t, err)
	assert.Eq(t, 1, len(entries))
	assert.Eq(t, "page.yaml", entries[0].Name())
}

func TestBackupFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.yaml")
	assert.NoErr(t, os.WriteFile(path, []byte("title: A\n"), 0o644))

	assert.NoErr(t, backupFile(path))

	got, err := os.ReadFile(path + ".bak")
	assert.NoErr(t, err)
	assert.Eq(t, "title: A\n", string(got))

	// 文件不存在时不应报错
	assert.NoErr(t, backupFile(filepath.Join(dir, "missing.yaml")))
}

func TestResolveWithinDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pages")

	t.Run("目录内路径通过", func(t *testing.T) {
		got, err := resolveWithinDir(dir, filepath.Join(dir, "home.yaml"))
		assert.NoErr(t, err)
		assert.Eq(t, "home.yaml", filepath.Base(got))
	})

	t.Run("子目录也通过", func(t *testing.T) {
		_, err := resolveWithinDir(dir, filepath.Join(dir, "sub", "home.yaml"))
		assert.NoErr(t, err)
	})

	t.Run("上级目录被拒绝", func(t *testing.T) {
		_, err := resolveWithinDir(dir, filepath.Join(dir, "..", "secret.yaml"))
		assert.Err(t, err)
	})

	t.Run("内嵌 .. 被拒绝", func(t *testing.T) {
		_, err := resolveWithinDir(dir, filepath.Join(dir, "a", "..", "..", "secret.yaml"))
		assert.Err(t, err)
	})
}

// 路径穿越的页面名不应被读到
func TestLoadPageConfigRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	// 在 pages 目录外放一个敏感文件
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "secret.yaml"), []byte("title: SECRET\n"), 0o644))

	pagesDir := filepath.Join(dir, "pages")
	assert.NoErr(t, os.MkdirAll(pagesDir, 0o755))

	mgr := &PageDataManager{PageDir: pagesDir, cacheMap: map[string]*PageConfig{}}

	_, err := mgr.LoadPageConfig("/a/../../secret")
	assert.Err(t, err)
}

// 空 items 的分组必须能插入第一个条目，且产出合法 YAML
func TestApplyBlockInsertIntoEmptyItems(t *testing.T) {
	tests := []struct {
		name    string
		content string
		indent  string
	}{
		{
			name:    "flow 空序列 items: []",
			content: "title: \"T\"\nservices:\n  - name: \"Empty\"\n    icon: \"fas fa-star\"\n    items: []\n",
			indent:  "      ",
		},
		{
			name:    "block 空序列 items:",
			content: "title: \"T\"\nservices:\n  - name: \"Empty\"\n    items:\n",
			indent:  "      ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := applyPageBlock([]byte(tt.content), BlockRequest{
				Kind:         BlockItem,
				Action:       BlockInsert,
				ServiceIndex: 0,
				YAML:         "name: \"First\"\nurl: \"https://first.example.com\"\n",
			})
			assert.NoErr(t, err)

			// 不能残留空的 flow 序列
			assert.False(t, containsLine(string(updated), "items: []"))
			// 缩进对齐到 items 的子级
			assert.True(t, containsLine(string(updated), tt.indent+"- name: \"First\""))
			assert.True(t, containsLine(string(updated), tt.indent+"  url: \"https://first.example.com\""))

			var cfg PageConfig
			assert.NoErr(t, yaml.Unmarshal(updated, &cfg))
			assert.Eq(t, 1, len(cfg.Services))
			assert.Eq(t, 1, len(cfg.Services[0].Items))
			assert.Eq(t, "First", cfg.Services[0].Items[0].Name)
		})
	}
}

func containsLine(text, line string) bool {
	for _, l := range splitLines([]byte(text)) {
		if l == line {
			return true
		}
	}
	return false
}
