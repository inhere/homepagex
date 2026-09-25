package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// writeFileAtomic 原子写入文件：先写同目录临时文件再 rename。
//
// 直接用 os.WriteFile 覆写，一旦进程/机器中途挂掉会留下半截 YAML，
// 下次启动解析必然失败。rename 在同一文件系统上是原子的。
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".hpx-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// 成功后已被 rename，这里的 Remove 会失败，忽略即可
	defer os.Remove(tmpName)

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}

// backupFile 把原文件备份为 {path}.bak（覆盖上一次备份）
func backupFile(path string) error {
	src, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	dst, err := os.Create(path + ".bak")
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Chmod(info.Mode().Perm())
}

// resolveWithinDir 校验 target 落在 dir 之内，返回清理后的绝对路径。
//
// 用于防御路径穿越：例如页面名里出现 "a/../../b" 时，filepath.Join 会把
// 路径清理到 pages 目录之外。
func resolveWithinDir(dir, target string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absDir, absTarget)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes base dir %q", target, dir)
	}
	return absTarget, nil
}
