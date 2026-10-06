package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/sysutil"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/homepagex/internal"
)

// resetOpts 清空所有选项。
// flag 只会设置出现在参数里的项，上一个用例留下的值会被带进下一个用例，所以每个用例开头都要重置。
func resetOpts() {
	gOpts = globalOpts{}
	serveOpts = serveCmdOpts{}
	initOpts = initCmdOpts{}
}

// newTestConfig 生成一份最小可用的配置与页面，返回配置文件路径
func newTestConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	assert.NoErr(t, os.MkdirAll(filepath.Join(dir, "pages"), 0o755))

	configYAML := `server:
  port: "18090"
  mode: release
pages_dir: "./pages"
frontend_dir: "./frontend/build"
auths:
  - "@*:rw"
`
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, defaultConfigFile), []byte(configYAML), 0o644))

	homeYAML := `title: "Test Home"
services:
  - name: "监控"
    items:
      - name: "Grafana"
        subtitle: "监控面板"
        tags: ["监控"]
        url: "https://grafana.example.com"
      - name: "Loki"
        subtitle: "日志聚合"
        url: "https://loki.example.com"
  - name: "其他"
    items:
      - name: "Portainer"
        url: "https://portainer.example.com"
`
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "pages", "home.yaml"), []byte(homeYAML), 0o644))
	return filepath.Join(dir, defaultConfigFile)
}

func TestConfigDirPrecedence(t *testing.T) {
	base := t.TempDir()

	t.Run("--config-dir 优先", func(t *testing.T) {
		resetOpts()
		t.Setenv(EnvConfigDir, filepath.Join(base, "env"))

		gOpts.ConfigDir = filepath.Join(base, "flag")
		dir, err := configDir()
		assert.NoErr(t, err)
		assert.Eq(t, filepath.Join(base, "flag"), dir)
	})

	t.Run("其次读环境变量", func(t *testing.T) {
		resetOpts()
		t.Setenv(EnvConfigDir, filepath.Join(base, "env"))

		dir, err := configDir()
		assert.NoErr(t, err)
		assert.Eq(t, filepath.Join(base, "env"), dir)
	})

	t.Run("最后回落到 ~/.config/homepagex", func(t *testing.T) {
		resetOpts()
		t.Setenv(EnvConfigDir, "")

		home, err := os.UserHomeDir()
		assert.NoErr(t, err)

		dir, err := configDir()
		assert.NoErr(t, err)
		assert.Eq(t, filepath.Join(home, ".config", "homepagex"), dir)
	})
}

func TestExpandUserDir(t *testing.T) {
	home, err := os.UserHomeDir()
	assert.NoErr(t, err)

	for _, tc := range []struct{ in, want string }{
		{"~", home},
		{"~/sub", filepath.Join(home, "sub")},
		{`~\sub`, filepath.Join(home, "sub")},
		{"/abs/path", "/abs/path"},
	} {
		got, err := expandUserDir(tc.in)
		assert.NoErr(t, err)
		assert.Eq(t, tc.want, got)
	}
}

func TestResolveConfigFile(t *testing.T) {
	configFile := newTestConfig(t)
	dir := filepath.Dir(configFile)

	t.Run("-c 指定的文件必须存在", func(t *testing.T) {
		resetOpts()
		gOpts.ConfigFile = configFile

		path, err := resolveConfigFile()
		assert.NoErr(t, err)
		assert.Eq(t, configFile, path)
	})

	t.Run("-c 指定的文件不存在时报错", func(t *testing.T) {
		resetOpts()
		gOpts.ConfigFile = filepath.Join(dir, "not-exist.yaml")

		_, err := resolveConfigFile()
		assert.Err(t, err)
	})

	t.Run("配置目录下的 config.yaml", func(t *testing.T) {
		resetOpts()
		gOpts.ConfigDir = dir

		path, err := resolveConfigFile()
		assert.NoErr(t, err)
		assert.Eq(t, configFile, path)
	})

	t.Run("环境变量指定的配置目录", func(t *testing.T) {
		resetOpts()
		t.Setenv(EnvConfigDir, dir)

		path, err := resolveConfigFile()
		assert.NoErr(t, err)
		assert.Eq(t, configFile, path)
	})

	t.Run("兜底用当前目录的 config.yaml", func(t *testing.T) {
		resetOpts()
		t.Setenv(EnvConfigDir, filepath.Join(dir, "empty-dir"))
		t.Chdir(dir)

		path, err := resolveConfigFile()
		assert.NoErr(t, err)
		assert.Eq(t, configFile, path)
	})
}

func TestLoadConfigAnchorsRelativeDirs(t *testing.T) {
	configFile := newTestConfig(t)
	resetOpts()
	gOpts.ConfigFile = configFile

	config, path, err := loadConfig()
	assert.NoErr(t, err)
	if err != nil {
		return
	}

	assert.Eq(t, configFile, path)

	// 相对路径必须锚定到配置文件所在目录，换成任意工作目录都能找到页面
	assert.Eq(t, filepath.Join(filepath.Dir(configFile), "pages"), config.PagesDir)
	assert.True(t, fsutil.IsDir(config.PagesDir))

	// 图标缓存目录同理：默认「配置文件所在目录/icons-cache」，而不是可执行文件旁边或进程 CWD
	assert.Eq(t, filepath.Join(filepath.Dir(configFile), "icons-cache"), config.IconCacheDir())
	assert.True(t, config.IconsRemote)
}

func TestInitCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cfg")
	resetOpts()

	assert.NoErr(t, newApp().RunWithArgs([]string{"init", dir}))

	// 生成的配置要能被自己加载，页面能被索引
	configFile := filepath.Join(dir, defaultConfigFile)
	config, err := internal.LoadConfig(configFile)
	assert.NoErr(t, err)
	if err != nil {
		return
	}
	assert.NoErr(t, config.ResolveRelativeDirs(configFile))

	sites, err := internal.IndexSites(config.PagesDir)
	assert.NoErr(t, err)
	assert.True(t, len(sites) > 0)

	// 再执行一次不会覆盖
	resetOpts()
	assert.NoErr(t, newApp().RunWithArgs([]string{"init", dir}))
}

func TestInitCommandGlobal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvConfigDir, dir)
	resetOpts()

	assert.NoErr(t, newApp().RunWithArgs([]string{"init", "-g"}))
	assert.True(t, fsutil.IsFile(filepath.Join(dir, defaultConfigFile)))

	// 全局配置就绪后，不带 -c 也能用 find
	resetOpts()
	assert.NoErr(t, newApp().RunWithArgs([]string{"find", "grafana"}))
}

func TestInitCommandRejectsGlobalWithDir(t *testing.T) {
	resetOpts()
	assert.Err(t, newApp().RunWithArgs([]string{"init", "-g", t.TempDir()}))
}

func TestFindCommand(t *testing.T) {
	configFile := newTestConfig(t)

	t.Run("命中单个站点", func(t *testing.T) {
		resetOpts()
		assert.NoErr(t, newApp().RunWithArgs([]string{"-c", configFile, "find", "grafana"}))
	})

	t.Run("中文与多关键词", func(t *testing.T) {
		resetOpts()
		assert.NoErr(t, newApp().RunWithArgs([]string{"-c", configFile, "find", "监控"}))
		resetOpts()
		assert.NoErr(t, newApp().RunWithArgs([]string{"-c", configFile, "find", "example.com", "loki"}))
	})

	t.Run("没有匹配时返回错误", func(t *testing.T) {
		resetOpts()
		assert.Err(t, newApp().RunWithArgs([]string{"-c", configFile, "find", "根本没有这个站点"}))
	})

	t.Run("缺少关键词时报错", func(t *testing.T) {
		resetOpts()
		assert.Err(t, newApp().RunWithArgs([]string{"-c", configFile, "find"}))
	})
}

func TestOpenCommand(t *testing.T) {
	configFile := newTestConfig(t)

	var opened []string
	openBrowser = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	t.Cleanup(func() { openBrowser = sysutil.OpenBrowser })

	t.Run("唯一匹配时打开浏览器", func(t *testing.T) {
		opened = nil
		resetOpts()

		assert.NoErr(t, newApp().RunWithArgs([]string{"-c", configFile, "open", "grafana"}))
		assert.Eq(t, []string{"https://grafana.example.com"}, opened)
	})

	t.Run("匹配到多个只列出，不打开", func(t *testing.T) {
		opened = nil
		resetOpts()

		assert.NoErr(t, newApp().RunWithArgs([]string{"-c", configFile, "open", "example.com"}))
		assert.Eq(t, 0, len(opened))
	})

	t.Run("没有匹配时报错，不打开", func(t *testing.T) {
		opened = nil
		resetOpts()

		assert.Err(t, newApp().RunWithArgs([]string{"-c", configFile, "open", "根本没有这个站点"}))
		assert.Eq(t, 0, len(opened))
	})
}

func TestPickFrontendDir(t *testing.T) {
	t.Run("配置的前端目录可用时原样返回", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoErr(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644))
		assert.Eq(t, dir, pickFrontendDir(dir))
	})

	// 兜底逻辑只会找「可执行文件旁边」，测试二进制的旁边没有前端，所以保持原值
	t.Run("没有前端时保持配置值", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "not-exist")
		assert.Eq(t, dir, pickFrontendDir(dir))
	})
}

func TestVersionCommand(t *testing.T) {
	resetOpts()
	assert.NoErr(t, newApp().RunWithArgs([]string{"-V"}))
}

func TestHelpWithoutArgs(t *testing.T) {
	resetOpts()
	assert.NoErr(t, newApp().RunWithArgs(nil))
}
