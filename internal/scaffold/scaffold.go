// Package scaffold 提供 `homepagex init` 用的示例配置与页面模板。
package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gookit/goutil/fsutil"
)

//go:embed config.yaml
//go:embed pages/*.yaml
var templateFS embed.FS

// Init 把示例配置与页面写入 dir 目录，返回实际写入和跳过的文件路径。
//
// force 为 false 时不会覆盖已存在的文件，而是记入 skipped —— 避免把用户改过的配置冲掉。
func Init(dir string, force bool) (written, skipped []string, err error) {
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create dir %q: %w", dir, err)
	}

	walkErr := fs.WalkDir(templateFS, ".", func(path string, ent fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if ent.IsDir() {
			return nil
		}

		target := filepath.Join(dir, filepath.FromSlash(path))
		if !force && fsutil.IsFile(target) {
			skipped = append(skipped, target)
			return nil
		}

		data, rerr := templateFS.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if merr := os.MkdirAll(filepath.Dir(target), 0o755); merr != nil {
			return merr
		}
		if werr2 := os.WriteFile(target, data, 0o644); werr2 != nil {
			return werr2
		}

		written = append(written, target)
		return nil
	})
	if walkErr != nil {
		return written, skipped, walkErr
	}
	return written, skipped, nil
}
