// Package frontend 提供内嵌的前端静态资源，用于单文件部署（二进制内自带前端）。
//
// 真实构建产物与占位资源由带构建标签的文件提供：
//   - embed_build.go（-tags embedfrontend）：内嵌 frontend/build，即 `pnpm run build` 的产物
//   - embed_placeholder.go（默认）：内嵌 frontend/placeholder 占位页
//
// 之所以要有占位版本：`frontend/build` 是构建产物、不进版本库，如果不带标签就
// `//go:embed all:build`，在没有前端产物的机器上 `go build ./...` / `go test ./...`
// 会因为「embed 匹配不到文件」直接编译失败。Makefile 的构建目标都会带上
// `-tags embedfrontend` 并保证先构建前端。
package frontend

import "io/fs"

// Assets 返回以「站点根」为基准的前端资源文件系统，路径形如
// "index.html"、"bundle.js"、"ajax/libs/font-awesome/6.4.0/css/all.min.css"。
//
// assetsFS / assetsRoot 由 embed_build.go 或 embed_placeholder.go 提供，
// 两者互斥（构建标签），任何一次编译都恰好有一个。
func Assets() fs.FS {
	sub, err := fs.Sub(assetsFS, assetsRoot)
	if err != nil {
		// 内嵌资源在编译期就已固定，这里失败说明内嵌目录结构被改坏了，属于编程错误
		panic("frontend: invalid embedded assets: " + err.Error())
	}
	return sub
}
