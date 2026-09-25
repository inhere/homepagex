# HomePageX

一个非常轻量的类似 Homer 的导航主页，使用 Go + Svelte 实现。

## 功能特性

- **YAML 配置**: 类似 Homer 的 YAML 格式定义页面链接
- **多页面支持**: `/` 对应 `home.yaml`，`/another` 对应 `another.yaml`
- **单块编辑**: 点某张卡片即可编辑这一个站点，**表单 / 源码可切换**，只改动相关行，保留注释与格式
- **视图切换**: 支持卡片视图和列表视图切换
- **内置过滤**: 支持按标题，描述，标签过滤服务项
- **权限体系**: 按路径的 `rw`/`ro`/`no` 权限、`!` 认证墙、`deny` 硬拒绝，页面编辑入口按权限显示
- **图标本地缓存**: `icons-local/...` 首次访问自动从 CDN 下载并缓存
- **FontAwesome 图标**: 支持 FontAwesome 图标
- **响应式设计**: 适配桌面和移动设备

## 效果预览

![Preview](deploy/preview.png)

## 快速开始

下载 Github release 最新版本:

```bash
wget https://github.com/inhere/go-homepagex/releases/latest/download/homepagex-linux-amd64
```

## 项目结构

```txt
go-homepagex/
├── internal/          # Go 后端服务
│   ├── config.go     # 配置加载
│   ├── page.go       # 页面配置解析
│   ├── auth.go       # Basic 认证
│   └── handlers.go   # HTTP 处理器
├── frontend/         # 前端应用
│   └── build/        # 构建输出
│       ├── index.html
│       └── app.js
├── pages/            # 页面 YAML 配置
│   ├── home.yaml     # 主页面配置
│   └── another.yaml
├── config.yaml   # 后端配置
├── main.go       # Go 入口文件
└── README.md
```

## 配置说明

### 后端配置 (config.yaml)

```yaml
server:
  port: "8090"
  session_ttl: "2h"   # 登录会话有效期

# 页面配置文件存放目录
pages_dir: "./pages"

# 前端构建目录
frontend_dir: "./frontend/build"
```

#### 认证配置

格式为 `{username}:{password}@{path:perm},{path2:perm2}`

- 使用 `@` 分隔账号和路径。账号部分留空（`@...`）表示**匿名规则**，构成「匿名基线」，即未登录用户能做什么。
- 使用 `:` 分隔用户名和密码；路径的权限后缀也用 `:`，取的是**最后一个** `:`。
- 使用 `,` 分隔多个路径。
- 权限后缀 `:rw`（读写）/ `:ro`（只读，可省略）/ `:no`（拒绝）。
- 路径写法：
  - `/a` —— 子树匹配，命中 `/a` 与 `/a/**`
  - `/a*` —— 前缀匹配，也会命中 `/ab`
  - `*`、`/*` —— 匹配全部路径
- 路径前缀 `!` 表示**认证墙**：该路径匿名不可访问，登录后按各自权限访问（对已登录用户不生效）。
- 顶层 `deny` 表示**硬拒绝**：任何人都不能访问（`admin` 也不例外），用于临时下线页面。

鉴权优先级（从高到低）：

1. `deny` 命中 → 拒绝
2. 写操作（POST）→ 必须已登录且具备 `rw`
3. 已登录 → 自身规则命中则使用（含显式 `:no`）
4. 已登录但自身规则未命中 → 回退匿名基线
5. 匿名 → 命中 `!` 认证墙则拒绝（需登录）
6. 没有任何规则命中 → 拒绝（fail closed）

> [!NOTE]
> 权限按页面配置，有页面的权限就有对应 api 的权限（api 访问去除 `/api/page` 前缀后检查）。

```yaml
auths:
  # admin 全站读写
  - admin:admin123@*:rw
  # user1 额外获得 /tools 的读写；其余读权限来自匿名基线，无需再写 /*:ro
  - user1:user123@/tools:rw
  # 匿名基线：全站只读，但 /inner* 需要登录
  - "@*,!/inner*"

# 任何人都不能访问（含 admin）
deny: []
```

常见写法：

- `admin:admin123@*:rw` —— admin 具有所有路径的完整权限。
- `user1:user123@/tools:rw` —— user1 对 `/tools` 读写，其他页面沿用它自己登录后的匿名基线（全站只读）。
- `@*` —— 所有路径公开只读。
- `@*,!/inner*` —— 全站公开只读，但 `/inner*` 需要登录。
- `user:pass@/secret:no,/tools:rw` —— user 对 `/secret` 显式拒绝，对 `/tools` 读写。
- `deny: ["/secret"]` —— `/secret` 对所有人（含已登录用户）一律拒绝。

> 只给特定用户开放某页面时，**不要**用 `@*` 这类宽泛规则，改成显式白名单，例如
> `@/:ro`、`@/tools:ro`，未列出的路径会按 fail closed 被拒绝。


### 页面配置 (pages/main.yaml)

```yaml
title: "Home Dashboard"
subtitle: "Welcome to your dashboard"
logo: "logo.png"

theme: "default"
color: "blue"
style: "cards"  # cards 或 list
columns: "3"

connectivity:
  check_interval: 30000
  mode: "ping"

services:
  - name: "Media"
    icon: "fas fa-play-circle"
    items:
      - name: "Plex"
        logo: "https://example.com/plex.png"
        subtitle: "Media server"
        tags: ["app"]
        url: "https://plex.example.com"
        target: "_blank"
```

## 页面配置规则

- `/` 路由对应 `pages/home.yaml`
- `/another` 路由对应 `pages/another.yaml`
- 依此类推: `/{name}` -> `/pages/{name}.yaml`

## 开发

### 1. 安装依赖

**Go (1.21+)**:
```bash
# 下载并安装 Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### 2. 编译后端

```bash
go mod tidy
go build -o homepagex
```

### 3. 运行

```bash
# 使用默认配置
./homepagex

# 或使用自定义配置文件
./homepagex /path/to/config.yaml
```

### 4. 访问

打开浏览器访问: http://localhost:8090

### 前端开发

前端使用 Svelte 框架。

#### 1. 安装依赖

```bash
cd frontend
pnpm install
```

#### 2. 开发模式

```bash
pnpm run dev
```

#### 3. 构建

```bash
pnpm run build
```

## 图标

支持 FontAwesome 图标:
- 使用 `fas fa-icon-name` 格式
- 完整图标列表: https://fontawesome.com/icons

### Dashboard ICONS

- https://github.com/homarr-labs/dashboard-icons
- https://selfh.st/icons/

## 许可证

MIT
