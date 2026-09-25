package internal

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func newTestPageManager(t *testing.T, files map[string]string) *PageDataManager {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		assert.NoErr(t, os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0o644))
	}

	return &PageDataManager{
		PageDir:  dir,
		cacheMap: make(map[string]*PageConfig),
	}
}

// 覆盖 P0-3：HTTP handler 并发访问缓存，不能出现 concurrent map write
func TestGetPageConfigConcurrent(t *testing.T) {
	mgr := newTestPageManager(t, map[string]string{
		"home":  "title: \"Home\"\n",
		"tools": "title: \"Tools\"\n",
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			name := "/home"
			if i%2 == 0 {
				name = "/tools"
			}

			// 交替走缓存与强制刷新，制造读写并发
			if _, err := mgr.GetPageConfig(name, i%3 == 0); err != nil {
				t.Errorf("GetPageConfig(%s): %v", name, err)
			}
			if i%5 == 0 {
				mgr.ClearCache(name)
			}
		}(i)
	}
	wg.Wait()
}

// 回归：GetPageConfig 与 ClearCache 曾使用不同的缓存 key，导致清除缓存无效
func TestClearCacheTakesEffect(t *testing.T) {
	mgr := newTestPageManager(t, map[string]string{
		"tools": "title: \"A\"\n",
	})

	page, err := mgr.GetPageConfig("/tools", false)
	assert.NoErr(t, err)
	assert.Eq(t, "A", page.Title)

	// 直接改文件：缓存命中时仍返回旧内容
	assert.NoErr(t, os.WriteFile(filepath.Join(mgr.PageDir, "tools.yaml"), []byte("title: \"B\"\n"), 0o644))
	cached, err := mgr.GetPageConfig("/tools", false)
	assert.NoErr(t, err)
	assert.Eq(t, "A", cached.Title)

	// 清除缓存后必须读到新内容
	mgr.ClearCache("/tools")
	fresh, err := mgr.GetPageConfig("/tools", false)
	assert.NoErr(t, err)
	assert.Eq(t, "B", fresh.Title)
}
