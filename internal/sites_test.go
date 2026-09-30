package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

// newSitesFixture 造一个页面目录，含两个页面 + 应被忽略的文件
func newSitesFixture(t *testing.T) string {
	t.Helper()

	pages := filepath.Join(t.TempDir(), "pages")
	assert.NoErr(t, os.MkdirAll(pages, 0o755))

	files := map[string]string{
		"home.yaml": `title: "Home"
services:
  - name: "监控"
    items:
      - name: "Grafana"
        subtitle: "监控面板"
        tags: ["监控", "metrics"]
        keywords: "dashboard"
        url: "https://grafana.example.com"
      - name: "缺失地址"
        subtitle: "没有 url 的条目"
  - name: "其他"
    items:
      - name: "Portainer"
        subtitle: "容器管理"
        url: "https://portainer.example.com"
`,
		"tools.yaml": `title: "Tools"
services:
  - name: "开发"
    items:
      - name: "Jenkins"
        tags: ["ci"]
        url: "https://jenkins.example.com"
`,
		// 下面两个都不应该进索引
		"home.local.yaml": "services:\n  - name: \"本地覆盖\"\n    items:\n      - name: \"Hidden\"\n        url: \"https://hidden.example.com\"\n",
		"notes.txt":       "not a page",
	}

	for name, content := range files {
		assert.NoErr(t, os.WriteFile(filepath.Join(pages, name), []byte(content), 0o644))
	}
	return pages
}

func TestIndexSites(t *testing.T) {
	sites, err := IndexSites(newSitesFixture(t))
	assert.NoErr(t, err)

	// 跳过 *.local.yaml、非 yaml 文件，以及没有 url 的条目
	assert.Eq(t, 3, len(sites))
	assert.Eq(t, "Grafana", sites[0].Name)
	assert.Eq(t, "Portainer", sites[1].Name)
	assert.Eq(t, "Jenkins", sites[2].Name)

	// 顺序为「页面文件名 -> 分组 -> 条目」，与页面里看到的一致
	assert.Eq(t, "home / 监控", sites[0].Location())
	assert.Eq(t, "home / 其他", sites[1].Location())
	assert.Eq(t, "tools / 开发", sites[2].Location())
}

func TestIndexSitesErrors(t *testing.T) {
	t.Run("目录不存在", func(t *testing.T) {
		_, err := IndexSites(filepath.Join(t.TempDir(), "nope"))
		assert.Err(t, err)
	})

	// 故意让页面配置出错时整体失败：索引不完整会让 open 把「两个匹配」看成
	// 「唯一匹配」而打开错误的站点
	t.Run("页面配置非法", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoErr(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("services: [oops"), 0o644))

		_, err := IndexSites(dir)
		assert.Err(t, err)
	})
}

func TestMatchSites(t *testing.T) {
	sites, err := IndexSites(newSitesFixture(t))
	assert.NoErr(t, err)

	t.Run("大小写不敏感", func(t *testing.T) {
		assert.Eq(t, 1, len(MatchSites(sites, "GRAFANA")))
	})

	t.Run("命中 url 与标签", func(t *testing.T) {
		assert.Eq(t, 3, len(MatchSites(sites, "example.com")))
		assert.Eq(t, 1, len(MatchSites(sites, "metrics")))
	})

	t.Run("命中中文副标题与分组名", func(t *testing.T) {
		assert.Eq(t, 1, len(MatchSites(sites, "监控面板")))
		assert.Eq(t, 1, len(MatchSites(sites, "监控")))
		assert.Eq(t, 1, len(MatchSites(sites, "开发")))
	})

	t.Run("命中页面里配置的 keywords", func(t *testing.T) {
		assert.Eq(t, 1, len(MatchSites(sites, "dashboard")))
	})

	t.Run("多个关键词需全部命中", func(t *testing.T) {
		assert.Eq(t, 1, len(MatchSites(sites, "grafana", "监控")))
		assert.Eq(t, 0, len(MatchSites(sites, "grafana", "jenkins")))
	})

	t.Run("单个参数里带空格也算多个关键词", func(t *testing.T) {
		assert.Eq(t, 1, len(MatchSites(sites, "grafana metrics")))
	})

	t.Run("空关键词不返回任何结果", func(t *testing.T) {
		assert.Eq(t, 0, len(MatchSites(sites, "")))
		assert.Eq(t, 0, len(MatchSites(sites)))
	})

	t.Run("无匹配", func(t *testing.T) {
		assert.Eq(t, 0, len(MatchSites(sites, "不存在的东西")))
	})
}
