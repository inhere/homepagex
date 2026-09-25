package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/gookit/goutil/fsutil"
)

// PageConfig 页面配置（类似 Homer 的格式）
type PageConfig struct {
	PageDefaults `yaml:",inline"`
	Connectivity Connectivity `yaml:"connectivity" json:"connectivity"`
	Services     []Service    `yaml:"services" json:"services"`
	Navs         []NavItem    `yaml:"navs" json:"navs"`

	// 内部设置，页面配置文件路径
	Pagefile string `yaml:"-" json:"-"`
}

// Connectivity 连接检查配置
type Connectivity struct {
	CheckInterval int    `yaml:"check_interval" json:"check_interval"`
	Mode          string `yaml:"mode" json:"mode"`
}

// Service 服务分组
type Service struct {
	Name  string `yaml:"name" json:"name"`
	Icon  string `yaml:"icon" json:"icon"`
	Items []Item `yaml:"items" json:"items"`
}

// Item 单个链接项
type Item struct {
	Name     string            `yaml:"name" json:"name"`
	Logo     string            `yaml:"logo" json:"logo"`
	Subtitle string            `yaml:"subtitle" json:"subtitle"`
	Tags     []string          `yaml:"tags" json:"tags"`
	Keywords string            `yaml:"keywords" json:"keywords"`
	URL      string            `yaml:"url" json:"url"`
	Target   string            `yaml:"target" json:"target"`
	Method   string            `yaml:"method" json:"method"`
	Headers  map[string]string `yaml:"headers" json:"headers"`
	Type     string            `yaml:"type" json:"type"`
}

// PageDataManager 页面数据管理器
type PageDataManager struct {
	Debug    bool
	PageDir  string
	Defaults PageDefaults
	// 默认导航项
	Navs []NavItem
	// 页面配置缓存 key is page name TODO 支持缓存过期
	// mu 保护 cacheMap：HTTP handler 天然并发访问
	mu       sync.RWMutex
	cacheMap map[string]*PageConfig
}

// PageDataMgr 页面数据管理器实例
var PageDataMgr *PageDataManager

// GetPageConfig 获取页面配置数据
func (m *PageDataManager) GetPageConfig(name string, refresh bool) (*PageConfig, error) {
	// 缓存 key 必须与 ClearCache 使用的 key 一致，否则保存后无法失效旧缓存
	key := m.getFilename(name)

	// feat: refresh=true 时跳过缓存，重新加载
	if !refresh {
		refresh = m.Debug
	}

	// 从缓存中获取
	if !refresh {
		m.mu.RLock()
		page, ok := m.cacheMap[key]
		m.mu.RUnlock()
		if ok {
			return page, nil
		}
	}

	page, err := m.LoadPageConfig(name)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if m.cacheMap == nil {
		m.cacheMap = make(map[string]*PageConfig)
	}
	m.cacheMap[key] = page
	m.mu.Unlock()

	return page, nil
}

const (
	// DefaultPageName 默认页面名
	DefaultPageName = "home"
	// DefaultPageFile 默认页面文件名
	DefaultPageFile = "home.yaml"
)

// getFilename 生成页面配置文件名
func (m *PageDataManager) getFilename(name string) string {
	// 移除开头的无效字符 /.
	name = strings.TrimLeft(name, "/.")
	if name == "" {
		return DefaultPageName
	}
	return name
}

// LoadPageConfig 加载页面配置
func (m *PageDataManager) LoadPageConfig(name string) (*PageConfig, error) {
	filename := m.getFilename(name)

	pagefile, err := resolveWithinDir(m.PageDir, filepath.Join(m.PageDir, filename+".yaml"))
	if err != nil {
		return nil, fmt.Errorf("invalid page name %q: %w", name, err)
	}

	var data []byte

	// debug mode 下，优先使用 {name}.local.yaml
	if m.Debug {
		dotLocalFile, lerr := resolveWithinDir(m.PageDir, filepath.Join(m.PageDir, filename+".local.yaml"))
		if lerr == nil && fsutil.IsFile(dotLocalFile) {
			pagefile = dotLocalFile
			data, _ = os.ReadFile(dotLocalFile)
		}
	}

	if len(data) == 0 {
		data, err = os.ReadFile(pagefile)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("page config not found: %s", filename)
		}
		return nil, fmt.Errorf("failed to read page config: %w", err)
	}

	page := &PageConfig{
		PageDefaults: m.Defaults,
	}
	if err := yaml.Unmarshal(data, page); err != nil {
		return nil, fmt.Errorf("failed to parse page config: %w", err)
	}

	// 记录页面配置文件路径
	page.Pagefile = pagefile

	// 设置默认值
	if page.Style == "" {
		page.Style = "cards"
	}
	if page.Columns == "" {
		page.Columns = "3"
	}

	// 如果页面没有配置 navs，使用默认配置
	if len(page.Navs) == 0 {
		page.Navs = m.Navs
	}
	return page, nil
}

// ClearCache 清除指定页面的缓存
func (m *PageDataManager) ClearCache(name string) {
	filename := m.getFilename(name)

	m.mu.Lock()
	delete(m.cacheMap, filename)
	m.mu.Unlock()
}
