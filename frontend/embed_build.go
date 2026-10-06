//go:build embedfrontend

package frontend

import "embed"

// 真实前端构建产物。
//
// 构建前必须先执行 `pnpm --dir frontend run build`（Makefile 的构建目标会保证）；
// 目录不存在时这里会直接编译失败并提示 `pattern all:build: no matching files found`，
// 而不是悄悄产出一个没有前端资源的二进制。
//
//go:embed all:build
var assetsFS embed.FS

// assetsRoot frontend/build 在 assetsFS 里的目录名
const assetsRoot = "build"

// Embedded 内嵌的是真实前端构建产物
const Embedded = true
