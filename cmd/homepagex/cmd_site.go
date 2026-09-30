package main

import (
	"fmt"
	"strings"

	"github.com/gookit/goutil/cflag/capp"
	"github.com/gookit/goutil/cliutil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/goutil/sysutil"
	"github.com/inhere/homepagex/internal"
)

// openBrowser 打开浏览器，抽成变量便于测试替换
var openBrowser = sysutil.OpenBrowser

func newFindCmd() *capp.Cmd {
	cmd := capp.NewCmd("find", "按关键词查找站点地址", runFind)
	cmd.Aliases = []string{"f"}
	cmd.OnAdd = func(c *capp.Cmd) {
		c.AddArg("keywords", "关键词，多个词需同时命中，如: homepagex find grafana 监控", true, nil, true)
	}
	return cmd
}

func newOpenCmd() *capp.Cmd {
	cmd := capp.NewCmd("open", "按关键词打开站点（唯一匹配时才打开浏览器）", runOpen)
	cmd.Aliases = []string{"o"}
	cmd.OnAdd = func(c *capp.Cmd) {
		c.AddArg("keywords", "关键词，匹配到多个时只列出不打开，如: homepagex open grafana", true, nil, true)
	}
	return cmd
}

func runFind(c *capp.Cmd) error {
	keywords := c.Arg("keywords").Strings()

	sites, err := loadSites()
	if err != nil {
		return err
	}

	matched := internal.MatchSites(sites, keywords...)
	if len(matched) == 0 {
		return fmt.Errorf("未找到匹配 %q 的站点（已索引 %d 个站点）", strings.Join(keywords, " "), len(sites))
	}

	cliutil.Infoln("匹配到", len(matched), "个站点:")
	printSites(matched)
	return nil
}

func runOpen(c *capp.Cmd) error {
	keywords := c.Arg("keywords").Strings()

	sites, err := loadSites()
	if err != nil {
		return err
	}

	matched := internal.MatchSites(sites, keywords...)
	switch len(matched) {
	case 0:
		return fmt.Errorf("未找到匹配 %q 的站点（已索引 %d 个站点）", strings.Join(keywords, " "), len(sites))
	case 1:
		site := matched[0]
		if err = openBrowser(site.URL); err != nil {
			return fmt.Errorf("打开浏览器失败 %s: %w", site.URL, err)
		}
		cliutil.Successln("已打开:", site.Name, "-", site.URL, "("+site.Location()+")")
		return nil
	default:
		// 命中多个就不猜，列出来让用户自己挑
		cliutil.Warnln("匹配到", len(matched), "个站点，未自动打开。请用更精确的关键词:")
		printSites(matched)
		return nil
	}
}

// loadSites 读取配置并建立站点索引
func loadSites() ([]internal.SiteEntry, error) {
	config, configPath, err := loadConfig()
	if err != nil {
		return nil, err
	}

	sites, err := internal.IndexSites(config.PagesDir)
	if err != nil {
		return nil, fmt.Errorf("%w\n       页面目录: %s\n       配置文件: %s", err, config.PagesDir, configPath)
	}
	if len(sites) == 0 {
		return nil, fmt.Errorf("没有索引到任何站点，请检查 %s 下的页面配置", config.PagesDir)
	}
	return sites, nil
}

// printSites 以表格列出站点。
//
// 自己排版而不用 cliutil.ShowTable：那个表格按 rune 数算宽度，
// 中文表头（名称/地址/位置）与中文站点名都会错位 —— 一个汉字占两列。
func printSites(sites []internal.SiteEntry) {
	rows := make([][]string, 0, len(sites)+1)
	rows = append(rows, []string{"名称", "地址", "位置"})
	for _, site := range sites {
		rows = append(rows, []string{site.Name, site.URL, site.Location()})
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if w := strutil.Utf8Width(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	for idx, row := range rows {
		var line strings.Builder
		for i, cell := range row {
			if i > 0 {
				line.WriteString("  ")
			}
			line.WriteString(cell)
			line.WriteString(strings.Repeat(" ", widths[i]-strutil.Utf8Width(cell)))
		}

		if idx == 0 {
			cliutil.Yellowln(line.String())
			continue
		}
		fmt.Println(line.String())
	}
}
