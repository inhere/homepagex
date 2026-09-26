# AGENTS.md

Guide for AI agents working in the HomePageX codebase.

- 优先使用中文回答和响应结果
- 过长的代码变动(超过30行)不要在终端完整输出，只输出前后部分即可
- 启动服务时，先 go build 再启动build后的程序，避免直接 go run 需要每次授权卡住

## Project Overview

HomePageX is a lightweight Homer-like dashboard homepage built with **Go (backend)** + **Svelte (frontend)**. It provides:

> 更多请查看 [project.md](docs/project.md)

### Full Development

1. Build frontend first: `cd frontend && pnpm run build`
2. Build & run backend: `make build && ./dist/homepagex`（或 `go run ./cmd/homepagex`）
3. Access at: `http://localhost:8090`

> 入口在 `cmd/homepagex/`，不在仓库根目录。

## Architecture

### Backend (Go)

- **Entry Point**: `cmd/homepagex/main.go` registers routes on `http.ServeMux`
- **Package**: All backend code is in `internal` package (imported as `github.com/inhere/homepagex/internal`)
- **Server**: Simple `http.ServeMux` based server (no framework)
- **Config**: YAML-based configuration with `goccy/go-yaml`
- **Auth**: 基于路径规则的权限系统，统一走 `Config.Resolve(username, path, isWrite)`（`internal/perm.go`）

### Frontend (Svelte)

- **Framework**: Svelte 4 with Rollup bundler
- **State Management**: Svelte stores (`writable`, custom persisted store)
- **Routing**: Client-side via `history.pushState` and `popstate` events
- **Styling**: Component-scoped CSS with CSS variables for theming

## Testing

- Go tests use standard `testing` package
- Assertions via `github.com/gookit/goutil/testutil/assert`
- Run with: `go test ./internal/...`
- Main test file: `internal/config_test.go` (auth parsing, path matching)

## Dependencies

### Go
- `goccy/go-yaml` - YAML parsing
- `gookit/goutil` - Utilities (strutil, fsutil, testutil)

### Frontend
- `svelte` - UI framework
- `rollup` with plugins - Bundler
- `sirv-cli` - Static file server (dev only)
