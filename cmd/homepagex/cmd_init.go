package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/cflag/capp"
	"github.com/gookit/goutil/cliutil"
	"github.com/inhere/homepagex/internal/scaffold"
)

// initCmdOpts init 命令选项
type initCmdOpts struct {
	// Global 写入全局配置目录
	Global bool
	// Force 覆盖已存在的文件
	Force bool
}

var initOpts initCmdOpts

func newInitCmd() *capp.Cmd {
	cmd := capp.NewCmd("init", "初始化示例配置与页面", runInit)
	cmd.OnAdd = func(c *capp.Cmd) {
		c.BoolVar(&initOpts.Global, "global", false, "初始化到全局配置目录，之后可直接 `homepagex serve`;false;g")
		c.BoolVar(&initOpts.Force, "force", false, "覆盖已存在的文件;false;f")
		c.AddArg("dir", "目标目录，默认当前目录", false)
	}
	return cmd
}

func runInit(c *capp.Cmd) error {
	dirArg := strings.TrimSpace(c.Arg("dir").String())
	if initOpts.Global && dirArg != "" {
		return fmt.Errorf("-g/--global 与目标目录不能同时指定")
	}

	var target string
	switch {
	case initOpts.Global:
		dir, err := configDir()
		if err != nil {
			return err
		}
		target = dir
	case dirArg != "":
		dir, err := expandUserDir(dirArg)
		if err != nil {
			return err
		}
		target = dir
	default:
		target = "."
	}

	target, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("无法解析目标目录: %w", err)
	}

	written, skipped, err := scaffold.Init(target, initOpts.Force)
	if err != nil {
		return err
	}

	for _, file := range written {
		cliutil.Successln("  已生成", file)
	}
	for _, file := range skipped {
		cliutil.Warnln("  已存在，跳过", file)
	}
	if len(skipped) > 0 && !initOpts.Force {
		cliutil.Infoln("（要覆盖已存在的文件，加 -f/--force）")
	}

	// 下一步提示
	// 注意：-c 是应用级选项，必须写在子命令前面（`serve -c` 会报 flag provided but not defined）
	configFile := filepath.Join(target, defaultConfigFile)
	if initOpts.Global {
		cliutil.Infoln("下一步: 运行 `homepagex serve`（将加载", configFile+"）")
	} else {
		cliutil.Infoln("下一步: 运行 `homepagex -c", configFile+" serve`")
	}

	if initOpts.Global {
		cliutil.Infoln("提示: 单文件部署无需前端目录，二进制内已内嵌前端资源；",
			"\n       源码运行（或想覆盖内嵌资源）时先执行 `pnpm --dir frontend run build`，再把配置里的 frontend_dir 指过去")
	}
	return nil
}
