package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// SiteEntry 站点索引里的一条记录，供 CLI 的 find / open 使用
type SiteEntry struct {
	// Name 站点名
	Name string
	// URL 站点地址
	URL string
	// Group 所属分组名
	Group string
	// Page 所在页面名（页面文件名去掉 .yaml）
	Page string
	// Subtitle 副标题
	Subtitle string
	// Tags 标签
	Tags []string
	// Keywords 页面里配置的额外关键词
	Keywords string
}

// SearchText 返回用于关键词匹配的文本（统一小写）
func (s SiteEntry) SearchText() string {
	parts := []string{s.Name, s.URL, s.Group, s.Page, s.Subtitle, s.Keywords}
	parts = append(parts, s.Tags...)
	return strings.ToLower(strings.Join(parts, " "))
}

// Location 站点在配置里的位置，如 "home / 监控"
func (s SiteEntry) Location() string {
	if s.Group == "" {
		return s.Page
	}
	return s.Page + " / " + s.Group
}

// IndexSites 扫描页面目录下所有页面配置，建立站点索引。
//
// 顺序为「页面文件名 → 分组顺序 → 条目顺序」，与页面里看到的顺序一致，
// 保证 find 的输出稳定可预期。
func IndexSites(pagesDir string) ([]SiteEntry, error) {
	list, err := os.ReadDir(pagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read pages dir %q: %w", pagesDir, err)
	}

	var sites []SiteEntry
	for _, ent := range list {
		if ent.IsDir() {
			continue
		}

		filename := ent.Name()
		// 只认 *.yaml；*.local.yaml 是 debug 下的本地覆盖文件，不算独立页面
		if !strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".local.yaml") {
			continue
		}

		pagefile, err := safeJoin(pagesDir, filename)
		if err != nil {
			return nil, err
		}

		data, err := os.ReadFile(pagefile)
		if err != nil {
			return nil, fmt.Errorf("failed to read page file %q: %w", pagefile, err)
		}

		page := &PageConfig{}
		if err = yaml.Unmarshal(data, page); err != nil {
			// 页面配置报错就整体失败：索引不完整会让 open 打开错误的站点
			// （本该两个匹配只剩一个，就被当成「唯一匹配」打开了）
			return nil, fmt.Errorf("failed to parse page file %q: %w", pagefile, err)
		}

		pageName := strings.TrimSuffix(filename, ".yaml")
		for _, svc := range page.Services {
			for _, item := range svc.Items {
				if strings.TrimSpace(item.URL) == "" {
					continue
				}

				sites = append(sites, SiteEntry{
					Name:     item.Name,
					URL:      item.URL,
					Group:    svc.Name,
					Page:     pageName,
					Subtitle: item.Subtitle,
					Tags:     item.Tags,
					Keywords: item.Keywords,
				})
			}
		}
	}

	return sites, nil
}

// MatchSites 按关键词过滤站点。
//
// 关键词以空白分隔，需全部命中才算匹配（AND）；匹配不区分大小写，
// 范围覆盖站点名、URL、副标题、标签、keywords、分组名与页面名。
func MatchSites(sites []SiteEntry, keywords ...string) []SiteEntry {
	var words []string
	for _, kw := range keywords {
		words = append(words, strings.Fields(strings.ToLower(kw))...)
	}
	if len(words) == 0 {
		return nil
	}

	var matched []SiteEntry
	for _, site := range sites {
		hay := site.SearchText()

		hit := true
		for _, word := range words {
			if !strings.Contains(hay, word) {
				hit = false
				break
			}
		}
		if hit {
			matched = append(matched, site)
		}
	}
	return matched
}
