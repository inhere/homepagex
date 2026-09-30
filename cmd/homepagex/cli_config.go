package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/cliutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/inhere/homepagex/internal"
)

// EnvConfigDir 配置目录的环境变量名
const EnvConfigDir = "HOMEPAGEX_CONFIG_DIR"

// defaultConfigFile 当前目录下的默认配置文件名
const defaultConfigFile = "config.yaml"

// configDir 返回配置目录，优先级：--config-dir > $HOMEPAGEX_CONFIG_DIR > ~/.config/homepagex
func configDir() (string, error) {
	if dir := strings.TrimSpace(gOpts.ConfigDir); dir != "" {
		return expandUserDir(dir)
	}
	if dir := strings.TrimSpace(os.Getenv(EnvConfigDir)); dir != "" {
		return expandUserDir(dir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法确定用户主目录: %w", err)
	}
	return filepath.Join(home, ".config", "homepagex"), nil
}

// expandUserDir 展开 ~ 与 ~/xxx
func expandUserDir(dir string) (string, error) {
	if dir != "~" && !strings.HasPrefix(dir, "~/") && !strings.HasPrefix(dir, `~\`) {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法展开 %q: %w", dir, err)
	}
	if len(dir) == 1 {
		return home, nil
	}
	return filepath.Join(home, dir[2:]), nil
}

// resolveConfigFile 决定本次使用哪个配置文件，返回的路径一定是已存在的文件。
//
// 查找顺序：-c 指定 > <配置目录>/config.yaml > 当前目录的 config.yaml。
// 最后一项是为了兼容发布包（配置与页面就跟在二进制旁边）和老的启动方式。
func resolveConfigFile() (string, error) {
	if file := strings.TrimSpace(gOpts.ConfigFile); file != "" {
		path, err := expandUserDir(file)
		if err != nil {
			return "", err
		}
		if !fsutil.IsFile(path) {
			return "", fmt.Errorf("配置文件不存在: %s", path)
		}
		return filepath.Abs(path)
	}

	dir, err := configDir()
	if err != nil {
		return "", err
	}

	global := filepath.Join(dir, defaultConfigFile)
	if fsutil.IsFile(global) {
		return global, nil
	}

	if fsutil.IsFile(defaultConfigFile) {
		cliutil.Infoln("提示: 未找到全局配置", global, "，改用当前目录的", defaultConfigFile)
		return filepath.Abs(defaultConfigFile)
	}

	return "", fmt.Errorf("未找到配置文件 %s，请先运行 `homepagex init -g` 生成示例配置，或用 -c 指定", global)
}

// loadConfig 解析配置文件，并把 pages_dir / frontend_dir 的相对路径锚定到配置文件所在目录。
func loadConfig() (*internal.Config, string, error) {
	path, err := resolveConfigFile()
	if err != nil {
		return nil, "", err
	}

	config, err := internal.LoadConfig(path)
	if err != nil {
		// config 为 nil 表示配置文件本身有问题（解析/校验失败），必须直接失败；
		// 非 nil 表示只是读不到文件（例如权限问题），有内置默认配置可兜底
		if config == nil {
			return nil, path, err
		}
		cliutil.Warnln("警告:", err)
	}

	if err = config.ResolveRelativeDirs(path); err != nil {
		return nil, path, err
	}
	return config, path, nil
}
