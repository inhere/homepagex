package internal

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/gookit/goutil/testutil/assert"
)

// blockTestYAML 带注释与空行的页面配置，用于验证按块编辑不会破坏其余内容
const blockTestYAML = `# Home Dashboard 主页面配置
title: "Home Dashboard"
subtitle: "Welcome"

services:
  - name: "Media" # 媒体分组
    icon: "fas fa-play-circle"
    items:
      - name: "Plex"
        url: "https://plex.example.com"
        tags: ["app"]

      - name: "Jellyfin"
        url: "https://jellyfin.example.com"

      - name: "Sonarr"
        url: "https://sonarr.example.com"

  - name: "Dev"
    icon: "fas fa-code"
    items:
      - name: "GitLab"
        url: "https://gitlab.example.com"
`

func listTestBlocks(t *testing.T) []BlockInfo {
	t.Helper()

	blocks, err := listPageBlocks([]byte(blockTestYAML))
	assert.NoErr(t, err)
	return blocks
}

func TestListPageBlocks(t *testing.T) {
	blocks := listTestBlocks(t)

	// 2 个分组 + 4 个条目
	assert.Eq(t, 6, len(blocks))

	svc0 := blocks[0]
	assert.Eq(t, BlockService, svc0.Kind)
	assert.Eq(t, 0, svc0.Index)
	assert.Eq(t, "Media", svc0.Name)
	assert.Eq(t, "fas fa-play-circle", svc0.Icon)

	// 分组块覆盖自身及其 items
	lines := splitLines([]byte(blockTestYAML))
	assert.Eq(t, "  - name: \"Media\" # 媒体分组", lines[svc0.LineRange[0]-1])
	assert.Eq(t, "        url: \"https://sonarr.example.com\"", lines[svc0.LineRange[1]-1])

	// 条目块
	var plex, jellyfin, sonarr BlockInfo
	for _, b := range blocks {
		if b.Kind != BlockItem || b.ServiceIndex != 0 {
			continue
		}
		switch b.Value.Name {
		case "Plex":
			plex = b
		case "Jellyfin":
			jellyfin = b
		case "Sonarr":
			sonarr = b
		}
	}

	assert.Eq(t, 0, plex.Index)
	assert.Eq(t, "https://plex.example.com", plex.Value.URL)
	assert.Eq(t, 1, len(plex.Value.Tags))
	assert.Eq(t, "app", plex.Value.Tags[0])

	// 每个条目行区间互不重叠，且 Jellyfin 前面的空行不属于任何条目
	assert.True(t, plex.LineRange[1] < jellyfin.LineRange[0])
	assert.Eq(t, "", strings.TrimSpace(lines[jellyfin.LineRange[0]-2]))
	assert.True(t, jellyfin.LineRange[1] < sonarr.LineRange[0])

	// 源码文本与行区间一致
	assert.Eq(t, "- name: \"Jellyfin\"\n        url: \"https://jellyfin.example.com\"",
		strings.TrimLeft(jellyfin.YAML, " "))
}

// 覆盖核心诉求：只改一个条目，其余行（含注释、空行、字段顺序）必须原样保留
func TestApplyBlockUpdateKeepsRestUntouched(t *testing.T) {
	before := splitLines([]byte(blockTestYAML))

	// 找到 Jellyfin
	var target BlockInfo
	for _, b := range listTestBlocks(t) {
		if b.Kind == BlockItem && b.Value != nil && b.Value.Name == "Jellyfin" {
			target = b
		}
	}
	assert.Eq(t, "Jellyfin", target.Value.Name)

	newBlock := "- name: \"Jellyfin\"\n  url: \"https://jellyfin.local:8096\"\n  subtitle: \"Media server\"\n"

	updated, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
		Kind:         BlockItem,
		Action:       BlockUpdate,
		ServiceIndex: target.ServiceIndex,
		Index:        target.Index,
		YAML:         newBlock,
	})
	assert.NoErr(t, err)

	after := splitLines(updated)
	start, end := target.LineRange[0], target.LineRange[1]

	// 目标块之前与之后的行必须逐行一致
	assert.Eq(t, strings.Join(before[:start-1], "\n"), strings.Join(after[:start-1], "\n"))
	assert.Eq(t, strings.Join(before[end:], "\n"), strings.Join(after[len(after)-(len(before)-end):], "\n"))

	// 注释仍在
	assert.True(t, strings.Contains(string(updated), "# Home Dashboard 主页面配置"))
	assert.True(t, strings.Contains(string(updated), `# 媒体分组`))

	// 其它条目未被改动
	assert.True(t, strings.Contains(string(updated), `name: "Plex"`))
	assert.True(t, strings.Contains(string(updated), `name: "Sonarr"`))
	assert.True(t, strings.Contains(string(updated), `name: "GitLab"`))

	// 改动后的配置可解析，且新值生效
	var cfg PageConfig
	assert.NoErr(t, yaml.Unmarshal(updated, &cfg))
	assert.Eq(t, 2, len(cfg.Services))
	assert.Eq(t, 3, len(cfg.Services[0].Items))
	assert.Eq(t, "https://jellyfin.local:8096", cfg.Services[0].Items[1].URL)
	assert.Eq(t, "Media server", cfg.Services[0].Items[1].Subtitle)
}

func TestApplyBlockInsert(t *testing.T) {
	updated, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
		Kind:         BlockItem,
		Action:       BlockInsert,
		ServiceIndex: 0,
		YAML:         "name: \"Overseerr\"\nurl: \"https://overseerr.example.com\"\n",
	})
	assert.NoErr(t, err)

	// 裸映射也应当被规范成序列元素，并保持缩进
	assert.True(t, strings.Contains(string(updated), `      - name: "Overseerr"`))
	assert.True(t, strings.Contains(string(updated), `        url: "https://overseerr.example.com"`))

	var cfg PageConfig
	assert.NoErr(t, yaml.Unmarshal(updated, &cfg))
	assert.Eq(t, 4, len(cfg.Services[0].Items))
	assert.Eq(t, 2, len(cfg.Services))
	// 插到末尾，不影响其它分组
	assert.Eq(t, "GitLab", cfg.Services[1].Items[0].Name)
	assert.Eq(t, "Overseerr", cfg.Services[0].Items[3].Name)

	// 原来的注释仍在
	assert.True(t, strings.Contains(string(updated), "# 媒体分组"))
}

func TestApplyBlockInsertService(t *testing.T) {
	updated, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
		Kind:   BlockService,
		Action: BlockInsert,
		YAML:   "- name: \"New\"\n  icon: \"fas fa-star\"\n  items:\n    - name: \"X\"\n      url: \"https://x.example.com\"\n",
	})
	assert.NoErr(t, err)

	var cfg PageConfig
	assert.NoErr(t, yaml.Unmarshal(updated, &cfg))
	assert.Eq(t, 3, len(cfg.Services))
	assert.Eq(t, "New", cfg.Services[2].Name)
	assert.Eq(t, "X", cfg.Services[2].Items[0].Name)
}

func TestApplyBlockDelete(t *testing.T) {
	var target BlockInfo
	for _, b := range listTestBlocks(t) {
		if b.Kind == BlockItem && b.Value != nil && b.Value.Name == "Jellyfin" {
			target = b
		}
	}

	updated, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
		Kind:         BlockItem,
		Action:       BlockDelete,
		ServiceIndex: target.ServiceIndex,
		Index:        target.Index,
	})
	assert.NoErr(t, err)

	s := string(updated)
	assert.False(t, strings.Contains(s, "Jellyfin"))
	assert.True(t, strings.Contains(s, `name: "Plex"`))
	assert.True(t, strings.Contains(s, `name: "Sonarr"`))

	// 不应留下连续空行
	assert.False(t, strings.Contains(s, "\n\n\n"))

	var cfg PageConfig
	assert.NoErr(t, yaml.Unmarshal(updated, &cfg))
	assert.Eq(t, 2, len(cfg.Services[0].Items))
	assert.Eq(t, "Sonarr", cfg.Services[0].Items[1].Name)
}

func TestApplyBlockErrors(t *testing.T) {
	t.Run("非法 YAML", func(t *testing.T) {
		_, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
			Kind: BlockItem, Action: BlockUpdate, ServiceIndex: 0, Index: 0,
			YAML: "name: [\n",
		})
		assert.Err(t, err)
	})

	t.Run("多个条目", func(t *testing.T) {
		_, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
			Kind: BlockItem, Action: BlockUpdate, ServiceIndex: 0, Index: 0,
			YAML: "- name: A\n- name: B\n",
		})
		assert.Err(t, err)
	})

	t.Run("空内容", func(t *testing.T) {
		_, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
			Kind: BlockItem, Action: BlockUpdate, ServiceIndex: 0, Index: 0,
			YAML: "   \n",
		})
		assert.Err(t, err)
	})

	t.Run("下标越界", func(t *testing.T) {
		_, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
			Kind: BlockItem, Action: BlockUpdate, ServiceIndex: 9, Index: 9,
			YAML: "- name: A\n",
		})
		assert.Err(t, err)
	})

	t.Run("未知动作", func(t *testing.T) {
		_, err := applyPageBlock([]byte(blockTestYAML), BlockRequest{
			Kind: BlockItem, Action: "nope", ServiceIndex: 0, Index: 0,
		})
		assert.Err(t, err)
	})
}

func TestNormalizeBlockYAML(t *testing.T) {
	tests := []struct {
		name   string
		kind   BlockKind
		indent int
		in     string
		want   string
	}{
		{
			name:   "裸映射补上序列标记并按目标缩进",
			kind:   BlockItem,
			indent: 6,
			in:     "name: A\nurl: https://a.example.com\n",
			want:   "      - name: A\n        url: https://a.example.com",
		},
		{
			name:   "已是序列元素时只调整缩进",
			kind:   BlockItem,
			indent: 6,
			in:     "- name: A\n  url: https://a.example.com\n",
			want:   "      - name: A\n        url: https://a.example.com",
		},
		{
			name:   "带多余前导缩进也统一归一到目标缩进",
			kind:   BlockItem,
			indent: 6,
			in:     "        - name: A\n          url: https://a.example.com\n",
			want:   "      - name: A\n        url: https://a.example.com",
		},
		{
			name:   "去掉尾随空行",
			kind:   BlockItem,
			indent: 6,
			in:     "- name: A\n\n\n",
			want:   "      - name: A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeBlockYAML(tt.kind, tt.in, tt.indent)
			assert.NoErr(t, err)
			assert.Eq(t, tt.want, got)
		})
	}
}
