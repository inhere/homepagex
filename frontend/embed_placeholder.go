//go:build !embedfrontend

package frontend

import "embed"

// 未带 embedfrontend 标签时内嵌的占位资源（frontend/placeholder）。
//
// 用途只有一个：让 `go build ./...` / `go test ./...` 在没有前端产物的机器上
// （比如只跑 Go 单测的 CI）也能编译通过。页面会提示前端未内嵌。
//
//go:embed all:placeholder
var assetsFS embed.FS

// assetsRoot frontend/placeholder 在 assetsFS 里的目录名
const assetsRoot = "placeholder"

// Embedded 内嵌的不是真实前端构建产物（只有占位页）
const Embedded = false
