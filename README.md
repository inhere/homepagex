# HomePageX

一个非常轻量的类似 Homer 的导航主页，使用 Go + Svelte 实现。

## 功能特性

- **YAML 配置**: 类似 Homer 的 YAML 格式定义页面链接
- **多页面支持**: `/` 对应 `home.yaml`，`/another` 对应 `another.yaml`
- **单块编辑**: 点某张卡片即可编辑这一个站点，**表单 / 源码可切换**，只改动相关行，保留注释与格式
- **视图切换**: 支持卡片视图和列表视图切换
- **内置过滤**: 支持按标题，描述，标签过滤服务项
- **权限体系**: 按路径的 `rw`/`ro`/`no` 权限、`!` 认证墙、`deny` 硬拒绝，页面编辑入口按权限显示
- **色彩模式**: 亮色 / 暗色 / 跟随系统（默认）三选一，与 6 个主题独立组合
- **图标本地缓存**: `icons-local/...` 首次访问自动从 CDN 下载并缓存（`icons_remote: false` 可完全离线）
- **FontAwesome 图标**: 支持 FontAwesome 图标
- **响应式设计**: 适配桌面和移动设备

## 效果预览

![Preview](deploy/preview.png)

## 快速开始

**使用 Eget 快速安装**

Can quickly install by [inherelab/eget](https://github.com/inherelab/eget)

```bash
eget install inhere/homepagex
```

下载 Github release 最新版本:

```bash
wget https://github.com/inhere/go-homepagex/releases/latest/download/homepagex-linux-amd64
chmod +x homepagex-linux-amd64

# 生成示例配置与页面（默认写到 ~/.config/homepagex），然后启动
./homepagex-linux-amd64 init -g
./homepagex-linux-amd64 serve
```

## 离线 / 内网单文件部署

前端资源已经内嵌进二进制（Makefile 的构建目标都会带上 `-tags embedfrontend`），
所以部署只需要 **二进制 + config.yaml + pages 目录**，不再需要 `frontend/build`。

```bash
# 1. 在有网的机器上构建（会先 `pnpm run build`，再把前端内嵌进二进制）
make build-linux                 # → dist/homepagex-linux-amd64

# 2. 拷到内网机器
scp dist/homepagex-linux-amd64 inner-host:/opt/homepagex/

# 3. 在内网机器上生成配置与页面，然后启动
ssh inner-host
cd /opt/homepagex
./homepagex-linux-amd64 init .                       # 生成 config.yaml + pages 示例
./homepagex-linux-amd64 -c /opt/homepagex/config.yaml serve
```

离线环境的配置要点：

```yaml
# 不访问任何外网地址：未缓存的图标直接 404，页面不会有 CDN 超时等待
icons_remote: false
# 图标缓存目录（相对配置文件所在目录），需要可写
icons_dir: "./icons-cache"
# 指向不存在的目录也可以，此时自动使用二进制内嵌的前端资源
frontend_dir: "./frontend/build"
```

- 启动日志里 `Frontend source:` 会明确告诉你是用磁盘目录还是内嵌资源。
- 想以后覆盖内嵌的前端，把 `frontend_dir` 指向一个含 `index.html` 的目录即可（目录优先）。
- 保持 `icons_remote: true` 也能用：下载失败的图标会被记住 10 分钟，
  不会出现「每次刷新都逐个图标等满 5s 超时」；但完全离线建议直接设为 `false`。
- systemd 示例：

```ini
[Unit]
Description=HomePageX dashboard
After=network.target

[Service]
WorkingDirectory=/opt/homepagex
ExecStart=/opt/homepagex/homepagex-linux-amd64 -c /opt/homepagex/config.yaml serve
Restart=on-failure
User=www-data

[Install]
WantedBy=multi-user.target
```

## 项目结构

```txt
homepagex/
├── cmd/homepagex/    # Go 入口（CLI 定义 + 路由注册）
├── internal/         # Go 后端服务
│   ├── config.go     # 配置加载与认证规则解析
│   ├── perm.go       # 权限模型（Resolve 统一鉴权）
│   ├── auth.go       # 登录会话与认证中间件
│   ├── page.go       # 页面配置解析与缓存
│   ├── blocks.go     # 按源码行区间做单块编辑
│   ├── file.go       # 原子写入 / 备份 / 路径校验
│   ├── handlers.go   # HTTP 处理器
│   ├── sites.go      # 站点索引与关键词匹配（find / open 用）
│   ├── frontend.go   # 前端资源来源：frontend_dir 优先，否则内嵌资源
│   ├── scaffold/     # init 用的示例配置与页面模板
│   └── util.go       # 工具函数（Content-Type、图标下载）
├── frontend/         # Svelte 前端
│   ├── build/        # 构建输出（-tags embedfrontend 时内嵌进二进制）
│   ├── placeholder/  # 未构建前端时的占位页（默认内嵌，保证 go build/test 可用）
│   └── assets.go     # 内嵌资源入口
├── pages/            # 页面 YAML 配置
├── deploy/           # Docker 部署文件
├── docs/             # 项目文档
├── Makefile          # 构建 / 交叉编译 / 发布
├── config.yaml       # 后端配置
└── README.md
```

## 配置说明

### 后端配置 (config.yaml)

```yaml
server:
  port: "8090"
  session_ttl: "2h"   # 登录会话有效期
  # 会话 cookie 的 Secure 属性：auto（默认，请求是 HTTPS 才带）/ true / false
  # 本地明文 HTTP 访问必须保持 auto 或 false，否则浏览器会丢弃 cookie、登录失效；
  # 部署在 TLS 终止的反向代理后面时请显式设为 true
  cookie_secure: auto

# 页面配置文件存放目录
pages_dir: "./pages"

# 前端目录：里面有 index.html 时优先用它（开发时改完 pnpm build 立即生效，也能覆盖内嵌资源）；
# 不存在或没有 index.html 时使用二进制内嵌的前端资源 —— 单文件部署无需该目录
frontend_dir: "./frontend/build"

# 图标缓存目录（相对配置文件所在目录）。
# 内嵌的前端资源是只读的，图标缓存必须写在这个可写目录里
icons_dir: "./icons-cache"

# 是否允许从 CDN 下载缺失的图标
#   true（默认）：缓存未命中时按 icons_cdn 下载并缓存
#   false：完全不访问外网（离线 / 内网部署），未缓存的图标直接返回 404
# 另外：下载失败的图标会被记住 10 分钟，期间不再重试，避免离线时每个请求都等满超时
icons_remote: true

# 图标 CDN 基础路径：icons-local/{key}/{type}/{name} → {value}{type}/{name}
icons_cdn:
  dashboard-icons: "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/"
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

**Go (1.24+)** 与 **Node + pnpm**（构建前端需要）:

```bash
# 安装 Go
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 安装 pnpm
npm install -g pnpm
```

### 2. 构建

推荐直接用 Makefile（会先构建前端并把前端内嵌进二进制，输出到 `dist/`）：

```bash
make build       # 构建当前平台（单文件，自带前端资源）
make build-all   # 交叉编译所有平台
make release     # 生成发布包（含二进制、pages、config.yaml）
```

也可以手动构建（注意 `-tags embedfrontend`，否则内嵌的是占位页）：

```bash
cd frontend && pnpm install && pnpm run build && cd ..
go mod tidy
go build -tags embedfrontend -o homepagex ./cmd/homepagex
```

### 3. 运行

```bash
# 生成示例配置与页面
./homepagex init -g          # 写到全局配置目录（~/.config/homepagex）
./homepagex init ./my-home   # 也可以写到指定目录

# 启动服务（默认加载 ~/.config/homepagex/config.yaml）
./homepagex serve
./homepagex -c ./my-home/config.yaml serve   # 指定配置文件（-c 是全局选项，要写在子命令前）
./homepagex serve --addr :9090               # 覆盖监听地址
./homepagex serve --mode debug               # 覆盖运行模式

# 按关键词找站点 / 打开站点
./homepagex find grafana       # 列出匹配的站点（多个关键词需同时命中）
./homepagex open grafana       # 唯一匹配才打开浏览器，匹配到多个只列出

# 查看版本与帮助
./homepagex -V
./homepagex -h
```

命令行选项：

**全局选项**（写在子命令前面）

| 选项 | 说明 |
|------|------|
| `-c`, `--config` | 配置文件路径（默认 `<配置目录>/config.yaml`） |
| `--config-dir` | 配置目录，默认 `$HOMEPAGEX_CONFIG_DIR` 或 `~/.config/homepagex` |
| `-V`, `-v`, `--version` | 打印版本后退出 |
| `-h`, `--help` | 打印帮助 |

**子命令**

| 命令 | 说明 |
|------|------|
| `serve`（别名 `s`） | 启动服务；`--addr` 覆盖 `server.port`，`--mode` 覆盖 `server.mode` |
| `init` | 生成示例配置与页面；`-g` 写到全局配置目录，`-f` 覆盖已存在文件，后可跟一个目标目录 |
| `find`（别名 `f`） | 按关键词列出匹配的站点 |
| `open`（别名 `o`） | 按关键词打开站点，唯一匹配时才打开浏览器 |

> 配置文件查找顺序：`-c` 指定 → `<配置目录>/config.yaml` → 当前目录的 `config.yaml`（兼容发布包布局，会打印提示）。
> 配置里的 `pages_dir` / `frontend_dir` 相对路径按**配置文件所在目录**解析，因此在任意目录下 `serve` 都能找到页面。
> 配置文件读不到时会回退到内置默认配置（匿名只读、`./pages`、`./frontend/build`）；
> 文件存在但内容非法（如权限后缀写错）会直接报错退出，不会静默带错配置启动。

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

#### 4. 代码检查

```bash
pnpm run lint       # 检查（CI 里也会跑）
pnpm run lint:fix   # 自动修复可修复项
```

> lint 里最关键的是 `no-undef`：像「用了某个 store 却忘了 import」这类问题
> Svelte 编译期不报错、构建也能通过，但会在浏览器里抛 ReferenceError。

## 图标

支持 FontAwesome 图标:
- 使用 `fas fa-icon-name` 格式
- 完整图标列表: https://fontawesome.com/icons

### Dashboard ICONS

- https://github.com/homarr-labs/dashboard-icons
- https://selfh.st/icons/

图标访问路径为 `icons-local/{cdn-key}/{type}/{name}`（如 `icons-local/dashboard-icons/png/plex.png`），
服务端缓存未命中时按 `icons_cdn` 下载并写入 `icons_dir`：

- `icons_remote: false` —— 不访问任何外网，未缓存的图标直接 404（离线/内网部署推荐）
- 下载失败的图标会被记住 10 分钟，期间不再重试，避免每个请求都等满下载超时
- 图标的搜索/选择依赖各 CDN 的元数据（在公网上）；离线时编辑弹窗会提示
  「图标元数据加载失败」，可直接手动填写图标路径，不影响保存

## 许可证

MIT
