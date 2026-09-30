package scaffold

import (
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/homepagex/internal"
)

func TestInit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cfg")

	written, skipped, err := Init(dir, false)
	assert.NoErr(t, err)
	assert.Eq(t, 0, len(skipped))
	assert.Eq(t, 3, len(written))

	// 模板必须是「自己就能跑」的配置：解析、相对目录、站点索引都要能过。
	// 这样模板改坏了会在这里失败，而不是等用户执行 init 之后才发现
	configFile := filepath.Join(dir, "config.yaml")
	assert.True(t, fsutil.IsFile(configFile))

	config, err := internal.LoadConfig(configFile)
	assert.NoErr(t, err)
	assert.NoErr(t, config.ResolveRelativeDirs(configFile))
	assert.Eq(t, filepath.Join(dir, "pages"), config.PagesDir)

	sites, err := internal.IndexSites(config.PagesDir)
	assert.NoErr(t, err)
	assert.True(t, len(sites) >= 6, "示例页面里的站点数量")

	// 默认不覆盖已有文件（避免冲掉用户改过的配置）
	written, skipped, err = Init(dir, false)
	assert.NoErr(t, err)
	assert.Eq(t, 0, len(written))
	assert.Eq(t, 3, len(skipped))

	// force 才覆盖
	written, skipped, err = Init(dir, true)
	assert.NoErr(t, err)
	assert.Eq(t, 3, len(written))
	assert.Eq(t, 0, len(skipped))
}
