
# Project Overview

HomePageX is a lightweight Homer-like dashboard homepage built with **Go (backend)** + **Svelte (frontend)**. It provides:
- YAML-based page configuration (similar to Homer)
- Multi-page support with path-based routing
- Basic authentication with path-based permissions
- Card and list view modes with tag filtering
- Icon caching from CDN
- YAML editor for authenticated users

## Commands

### Backend (Go)

```bash
# Install dependencies
go mod tidy

# Run the server (uses config.yaml by default)
go run main.go

# Run with custom config
go run main.go /path/to/config.yaml

# Build binary
go build -o homepagex

# Run tests
go test ./internal/...

# Run specific test
go test -run TestParseAuths ./internal/...
```

### Frontend (Svelte)

```bash
cd frontend

# Install dependencies (uses pnpm)
pnpm install

# Development mode with hot reload
pnpm run dev

# Build for production
pnpm run build

# Serve built files locally
pnpm run start
```

### Full Development

1. Build frontend first: `cd frontend && pnpm run build`
2. Run backend: `cd .. && go run main.go`
3. Access at: `http://localhost:8090`

## Project Structure

```
go-homepagex/
├── main.go              # Go entry point, route registration
├── config.yaml          # Main server configuration
├── go.mod / go.sum      # Go dependencies
├── internal/            # Backend code (internal package)
│   ├── config.go        # Config loading and auth parsing
│   ├── config_test.go   # Unit tests for config
│   ├── auth.go          # Basic auth middleware
│   ├── handlers.go      # HTTP handlers (API endpoints)
│   ├── page.go          # Page config parsing and caching
│   ├── server.go        # Server struct and utilities
│   ├── types.go         # DTO types (LoginInfo, PageDataResponse)
│   ├── init.go          # PageDataManager initialization
│   └── util.go          # Helper functions (content-type, icon download)
├── frontend/            # Svelte frontend
│   ├── src/
│   │   ├── main.js      # Entry point
│   │   ├── App.svelte   # Root component
│   │   ├── stores.js    # Svelte stores (state management)
│   │   └── components/  # Svelte components
│   ├── public/          # Static assets
│   ├── build/           # Build output (served by Go)
│   ├── package.json     # npm dependencies
│   └── rollup.config.js # Build configuration
├── pages/               # Page YAML configurations
│   ├── home.yaml        # Maps to route /
│   ├── tools.yaml       # Maps to route /tools
│   └── ...              # /{name} -> pages/{name}.yaml
└── deploy/              # Docker deployment files
    ├── Dockerfile
    └── docker-compose.yml
```

## Architecture

### Backend (Go)

- **Entry Point**: `main.go` registers routes on `http.ServeMux`
- **Package**: All backend code is in `internal` package (imported as `github.com/inhere/homepagex/internal`)
- **Server**: Simple `http.ServeMux` based server (no framework)
- **Config**: YAML-based configuration with `goccy/go-yaml`
- **Auth**: Custom Basic Auth with path-based permission system

### Frontend (Svelte)

- **Framework**: Svelte 4 with Rollup bundler
- **State Management**: Svelte stores (`writable`, custom persisted store)
- **Routing**: Client-side via `history.pushState` and `popstate` events
- **Styling**: Component-scoped CSS with CSS variables for theming

## Key Patterns

### Go Backend Patterns

1. **Config Loading** (`internal/config.go`):
   ```go
   config, err := internal.LoadConfig("config.yaml")
   internal.Init(config)
   server := internal.NewServer(config)
   ```

2. **Handler Pattern** (`internal/handlers.go`):
   - Handlers are methods on `Server` struct
   - Use `sendJSON()` and `sendError()` helpers
   - API responses wrapped in `APIResponse{Success, Data, Error}`

3. **Auth Middleware** (`internal/auth.go` + `internal/perm.go`):
   - `BasicAuthMiddleware()` 统一调用 `Config.Resolve(username, path, isWrite)`
   - 用户名注入 `r.Context()` 的 `ContextKeyUsername`
   - 权限值：`rw`（读写）/ `ro`（只读）/ `no`（拒绝）；规则最具体者优先，同级取更严格
   - `!path` 是「认证墙」（仅对匿名生效）；顶层 `deny` 是任何人都不能访问
   - 未命中任何规则 → 拒绝（fail closed）

4. **Page Manager** (`internal/page.go`):
   - `PageDataMgr` is a global singleton
   - Caches parsed page configs in memory
   - Debug mode loads `{name}.local.yaml` preferentially

### Svelte Frontend Patterns

1. **State Management** (`stores.js`):
   ```javascript
   import { pageConfig, currentRoute, viewStyle, userInfo } from './stores.js';
   // Use with $ prefix: $pageConfig, $currentRoute
   ```

2. **API Calls** (`App.svelte`):
   ```javascript
   const response = await fetch(`/api/page${route}`);
   const result = await response.json();
   if (result.success) {
     pageConfig.set(result.data);
   }
   ```

3. **Component Communication**:
   - Parent-child: props
   - Child-parent: `createEventDispatcher()` with `on:event` handlers

## Configuration

### Main Config (`config.yaml`)

```yaml
server:
  port: "8090"
  mode: debug  # debug mode: loads .local.yaml files, skips cache

# Auth format: user:pass@path:perm,path2:perm2
# @... = 匿名规则（匿名基线）；:rw = 读写；:ro = 只读(默认)；:no = 拒绝
# !path = 认证墙（仅匿名不可访问，登录后即可见）
auths:
  - admin:admin123@*:rw
  # 读权限来自下面的匿名基线，无需再写 /*:ro
  - user1:user123@/tools:rw
  - "@*,!/inner*"

# 硬拒绝：任何人都不能访问（含 admin）
deny: []

pages_dir: "./pages"
frontend_dir: "./frontend/build"

page_defaults:
  theme: "ocean-depths"

page_navs:
  - name: "Home"
    icon: "fas fa-home"
    url: "/"
```

### Page Config (`pages/*.yaml`)

```yaml
title: "Dashboard"
subtitle: "Welcome"
theme: "ocean-depths"  # 6 themes available
style: "cards"         # cards or list
columns: "3"

connectivity:
  check_interval: 30000
  mode: "ping"

services:
  - name: "Media"
    icon: "fas fa-play-circle"
    items:
      - name: "Plex"
        logo: "icons-local/dashboard-icons/png/plex.png"
        subtitle: "Media server"
        tags: ["app"]
        url: "https://plex.example.com"
        target: "_blank"
```

## Routing

- `/` → `pages/home.yaml`
- `/tools` → `pages/tools.yaml`
- `/inner-tools` → `pages/inner-tools.yaml`
- Pattern: `/{name}` → `pages/{name}.yaml`

## API Endpoints

实际注册的路由见 `main.go`（没有 `/api/health`、`/api/auth`）：

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/page[/{path}]` | Optional | 获取页面配置（导航按权限过滤，含 `can_write`） |
| `GET /api/page[/{path}]?op=r` | ro | 获取整份原始 YAML |
| `GET /api/page[/{path}]?op=blocks` | ro | 列出可单独编辑的块（分组 / 条目）及其源码行区间 |
| `POST /api/page[/{path}]?op=w` | rw | 保存整份 YAML（先备份再原子写入） |
| `POST /api/page[/{path}]?op=block` | rw | 修改单个块（`update` / `insert` / `delete`），只替换相关行 |
| `POST /api/login` | No | UI 登录，成功设置会话 cookie |
| `POST /api/logout` | No | 退出登录，清理会话与 cookie |
| `GET /icons-local/{path}` | No | 图标本地缓存（缺失时按 `icons_cdn` 下载） |

> 权限按页面路径判定：请求 `/api/page` 时会去掉该前缀再走 `Config.Resolve`。
> 写操作（非 GET）一律要求已登录且具备 `rw`。

## Icon System

- Icons use FontAwesome: `fas fa-icon-name`
- Logo images can use CDN URLs or local cache
- `icons-local/dashboard-icons/png/plex.png` → cached from CDN defined in `icons_cdn`

## Themes

Available themes (defined in `stores.js`):

1. `ocean-depths` - 海洋深处 (default)
2. `tech-innovation` - 科技创新
3. `modern-minimalist` - 现代极简
4. `midnight-galaxy` - 午夜星河
5. `forest-canopy` - 森林树冠
6. `arctic-frost` - 北极冰霜

主题只提供色值映射，组件一律使用**语义 token**（`--bg-deep` / `--ink` / `--accent` / `--surface` / `--border` 等），
由 `stores.js::getThemeTokens()` 生成后在 `App.svelte` 注入。不要在组件里直接使用主题背景色当文字色。

## Gotchas

1. **Build frontend before running backend**: The Go server serves `frontend/build/`, which must exist.

2. **Auth middleware context**: Username is stored in `r.Context().Value(ContextKeyUsername)`, not in request headers.

3. **Page cache**: In production mode, page configs are cached. Use `?refresh=true` to bypass cache.

4. **Debug mode**: `mode: debug` in config enables:
   - Preferential loading of `.local.yaml` files
   - Skipping page cache

5. **Permissions are path-based**: The API paths `/api/page/*` are checked against page paths (prefix stripped).

6. **Frontend routing**: Uses `history.pushState` - all routes fall back to `index.html` on the backend.

7. ***.local.yaml files**: Listed in `.gitignore`, used for local development overrides.

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
