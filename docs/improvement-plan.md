# HomePageX 改进方案（权限 / 页面编辑 / 样式）

> **日期：2026-09-25**
>
> 本文档基于一次代码走查整理。
> 目标：把「权限设计、页面编辑、页面样式」三块的问题按优先级拆成可独立落地的任务，每条给出具体文件、改法、测试与验收标准。
> 走查范围：`main.go`、`internal/*.go`、`frontend/src/**`、`pages/*.yaml`、`config.yaml`、`deploy/*`、`README.md`、`project.md`。

---

## 0. 实施进度（滚动更新）

| 批次 | 状态 | 备注 |
| --- | --- | --- |
| 第 1 批 P0（正确性与安全） | ✅ 已完成（2026-09-25） | P0-1 / P0-2 / P0-3 已修复，新增 `perm_test.go`、`auth_test.go`、`page_test.go`、`handlers_test.go`；`go build` / `go vet` / `go test` 全绿，并已对真实服务做端到端验证 |
| 第 2 批 权限模型收敛 | ✅ 已完成（2026-09-25） | 新增 `internal/perm.go` 统一 `Resolve`；`!` 明确为认证墙；新增顶层 `deny`；支持 `:no`；下发 `can_write` 并删除前端重复匹配逻辑；`MatchAuthConfig` 已删除。**配置模型有变化，见 §0.3** |
| 第 3 批 单块编辑（表单 ↔ 源码） | ✅ 已完成（2026-09-25） | 新增 `internal/file.go`（原子写 + 备份 + 路径穿越防护）与 `internal/blocks.go`（行区间块 API）；前端新增 `BlockEditor.svelte`；图标插入改走 `icons-local`；整文件编辑补脏检查 / Ctrl+S。见 §0.4 |
| 第 4 批 样式与主题重构 | ✅ 已完成（2026-09-25） | 拆出语义色 token（`--ink` / `--accent` / `--surface` / `--border` 等）并替换全部「深色背景色当前景色」的用法；收敛为两段深色渐变；响应式列数改由 `--columns` 参与计算；移动端标签栏可横向滚动。见 §0.5 |
| 第 5 批 工程收尾 | ✅ 已完成（2026-09-25） | 删除 `internal/service` 空壳；补齐 `getContentType`；静态请求日志降到 debug；重写 Dockerfile / docker-compose；README / project.md 同步；`pages/*.yaml` 图标改走 `icons-local`（含 2 个名称纠正）。见 §0.5 |

### 0.1 第 1 批实际改动清单

| 文件 | 改动 |
| --- | --- |
| `internal/handlers.go` | YAML 校验失败改用 `sendError(..., 400)` + `yaml.FormatError(err, false, true)`（带行号与源码片段）；`handlePagePost` 的未登录分支改为 JSON 错误 |
| `internal/page.go` | `PageDataManager` 增加 `sync.RWMutex`；缓存 key 统一为 `getFilename(name)`，修复 `ClearCache` 因 key 不一致而失效的问题 |
| `internal/init.go` | 初始化 `cacheMap`，避免懒初始化在并发下的重复建 map |
| `internal/config.go` | 新增 `GuestBaselinePerm`（匿名允许基线，忽略 `!` 认证墙）；新增 `parsedAuthOrder` 并按声明顺序遍历 `MatchAuthConfig` |
| `internal/auth.go` | 新增 `userPermForPath`：用户规则未命中时回退匿名基线，修复「登录后反而 403」 |
| `frontend/src/components/YamlEditor.svelte` | 保存改为同时校验 HTTP 状态与 `payload.success`；去掉保存后的重复关闭 |
| 新增 4 个测试文件 | 覆盖 YAML 校验 400、权限回退、认证墙、fail-closed、并发缓存、缓存失效 |

> 端到端实测（release 模式 + 自定义 auths）：匿名 `GET /`=200、匿名 `GET /inner-tools`=401、匿名写=401；`user2`（无 `/*:ro` 兜底）登录后 `GET /`=**200（修复前 403）**、`GET /inner-tools`=200、写只读路径=403；`user1` 对 `/tools` 提交非法 YAML=**400** 且文件未被改写。

### 0.2 实施中新发现的问题

- **`MatchAuthConfig` 结果不确定（测试 flaky）**：该函数遍历 `parsedAuths`（Go map），而 map 的遍历顺序是随机的，所以「第一个匹配者胜」的结果不稳定。`TestMatchAuthConfig/多用户配置-匹配到第一个` 因此约 **1/6 概率失败**（已在未修改的 HEAD `b73396e` 上用独立 worktree 复现）。这正是 §3.7「测试给假信心」的实例。第 2 批删除该函数时该测试也一并移除，改为 `Resolve` 的表驱动测试。

### 0.3 第 2 批实际改动清单（配置模型有变化）

**代码**

| 文件 | 改动 |
| --- | --- |
| `internal/perm.go`（新增） | `Perm` / `pathRule` / `Access`；规则归一化（`normalizePattern`、`parseRuleToken`）；`bestMatch`（最具体优先，同级取更严格）；统一的 `Resolve(username, path, isWrite)`；`IsNeedAuth` 与 `FilterNavsByPermission` 改为基于 `Resolve` |
| `internal/config.go` | 新增 `Deny` 字段与 `hardDenyRules` / `guestRules` / `guestDenyRules`；`parseAuths` 重写为归一化规则并**在配置写错时启动失败**；删除 `MatchUserAuthConfig`、`MatchAuthConfig`、`pathMatch`、`GuestBaselinePerm`、`IsNeedAuth`（迁到 perm.go）；`AuthConfig` 由 `PathPerms []string` 改为 `Rules []*pathRule`，去掉运行期 `Permission` 字段 |
| `internal/auth.go` | 中间件收敛为一次 `Resolve` 调用，未登录 401、已登录无权 403 |
| `internal/handlers.go` | `handlePageGet` 下发 `can_write`；`handlePagePost` 用 `Resolve().CanWrite` 做纵深防御 |
| `internal/types.go` | `PageDataResponse` 增加 `can_write` |
| `frontend/src/components/Toolbar.svelte` | 删除 JS 复刻的 `matchPermForPath` / `hasWritePermissionForRoute`，改用 `$pageConfig.can_write` |
| `config.yaml` / `README.md` | 认证配置说明重写：`!` = 认证墙、`deny` = 硬拒绝、写操作需登录、匿名基线回退，并给出优先级列表 |

**配置模型变化（升级时需注意）**

1. `!path` 现在明确是「认证墙」：**只对匿名生效**，登录后即可按自身权限访问。语义与之前一致，但现在是文档化的契约并被测试固化。
2. 新增顶层 `deny: [path]`：**无条件硬拒绝**，`admin` 也不例外，用于临时下线页面。
3. `:no` 现在真正生效（此前 `"/a:no"` 会被解析成 `pattern=/a, perm="no:ro"` → 读反而放行）。
4. **用户规则不再需要 `/*:ro` 兜底**：登录后自动继承匿名基线。`config.yaml` 里的 `user1:user123@/tools:rw,/*:ro` 已简化为 `user1:user123@/tools:rw`（行为不变，但不再有「不写就 403 / 写了就等于全站开读」的两难）。
5. 路径写法统一：`/a` 子树、`/a*` 前缀、`/a/**` 子树、`*` / `/*` / `/**` 全匹配。旧写法含义不变。
6. 权限后缀只取**最后一个** `:`，且无法识别的后缀会让启动失败（fail fast），不再静默降级。

> 端到端实测（release 模式，auths = `admin@*:rw` + `user1@/tools:rw` + `@*,!/inner*`，`deny: [/another]`）：
> 匿名 `GET /`=200、`GET /inner-tools`=401、`GET /another`=401、`POST /tools`=401、`can_write`=false；
> `user1` `GET /`=200、`GET /inner-tools`=**200（越过认证墙）**、`GET /another`=**403（deny 对已登录同样生效）**、`POST /tools` 非法 YAML=400、`POST /`=403、`/tools` 的 `can_write`=true；
> `admin` `POST /another`=**403（deny 优先于 `*:rw`）**；
> 导航过滤：匿名 `Home, Tools, Ext`；`user1` `Home, Tools, InnerTools, Ext`。
>
> `go test -race -count=3 ./internal/...` 已在 `golang:1.25` 容器内通过（宿主无 gcc，故走 Docker；容器内挂载宿主 `GOMODCACHE` 并设 `GOPROXY=off`）。

### 0.4 第 3 批实际改动清单（单块编辑）

**后端**

| 文件 | 改动 |
| --- | --- |
| `internal/file.go`（新增） | `writeFileAtomic`（临时文件 + rename，避免写一半损坏配置）、`backupFile`（写入前备份为 `{name}.yaml.bak`）、`resolveWithinDir`（路径穿越防护） |
| `internal/blocks.go`（新增） | `BlockInfo` / `BlockRequest`；`listPageBlocks` 用 `goccy/go-yaml` 的 AST（`SequenceEntryNode.Start` 取起始行、递归求子树最大行号取结束行）算出每块的**源码行区间**；`applyPageBlock` 对行区间做**文本级替换 / 插入 / 删除**，其余内容原样保留；`normalizeBlockYAML` 统一缩进并补 `- ` 标记；`insertIntoEmptyItems` 处理 `items: []` 的首个条目插入 |
| `internal/handlers.go` | `PageApiHandler` 改为按 `op` 分发，新增 `GET ?op=blocks`、`POST ?op=block`；保存路径统一改为「备份 → 原子写入」；`?op=r` 不再返回服务器绝对路径 |
| `internal/page.go` | `LoadPageConfig` 用 `resolveWithinDir` 校验解析后的路径确实落在 `pages_dir` 内 |
| `internal/config.go` / `types.go` | 新增 `IconCDNKeys()`，随页面数据下发 `icon_cdn_keys` |

**前端**

| 文件 | 改动 |
| --- | --- |
| `BlockEditor.svelte`（新增） | 单卡片 / 单分组编辑弹窗，**表单 ↔ 源码**可切换；表单保存时只替换块内涉及的行（`patchBlockText`），未改动的字段、行尾注释、缩进都保留；源码保存时按用户文本替换；支持删除、Ctrl+S、Esc、未保存关闭确认 |
| `App.svelte` | 过滤数据时保留 `_serviceIndex` / `_itemIndex`，保证搜索、标签筛选后仍能定位到文件里的原始位置；接入 `BlockEditor`；卡片删除走块 API |
| `ServiceItem.svelte` | hover 显示「编辑 / 删除」按钮；**顺带修掉列表视图 `.url-popover`**（原来是绝对定位且 `top` 被注释掉，导致 URL 永久浮在行上盖住标题），改为常态化内联 + 超长省略 |
| `ServiceGroup.svelte` | 分组标题 hover 显示「编辑分组 / 新增站点」 |
| `Toolbar.svelte` | 「编辑」改为「原始 YAML」（高级入口），新增「新增分组」 |
| `YamlEditor.svelte` | 补脏检查（关闭前确认）、Ctrl+S、保存中禁止关闭；`bind:this` 取代 `document.querySelector` |
| `IconSearch.svelte` | 由后端下发的 `icon_cdn_keys` 决定可选图标源；插入值改为 `icons-local/{key}/png/{name}.png`（走本地缓存，不再直连 CDN）；去掉 `brightness(0) invert(1)`，预览恢复原色 |
| `package.json` | 新增 `js-yaml`（源码 ↔ 表单双向同步需要解析 YAML） |

**关键取舍**

1. **不做整文件 JSON 往返**：块编辑采用「定位源码行区间 → 只替换那几行」，因此注释、空行、字段顺序、其它条目**完全不受影响**。实测只改一个条目的 URL 时，`git diff` 恰好只有那 2 行，文件总行数不变。
2. 表单保存用「客户端在原始块文本上打补丁」而不是重新序列化整个块，所以即使只改一个字段，也不会把用户原有的 `tags: ["app"]` 这种写法改成块序列。
3. 空分组（`items: []`）也能插入首个条目（会先替换掉那一行，否则会产出非法 YAML）。
4. 整文件编辑器降级为高级入口并保留兜底能力；块定位失败时后端返回明确提示，引导用户改用整文件编辑。

**验证**

- `go test ./internal/...` 全绿（新增 `blocks_test.go`、`file_test.go`，含「只改目标块、其余逐行不变」的断言）；`go vet` 干净；`go test -race -count=2` 在 `golang:1.25` 容器内通过。
- 端到端（release 模式 + **临时 pages 目录**，避免污染仓库文件）：
  `GET ?op=blocks` 正确列出 15 个块及行区间；更新一个条目 → **103 行不变、仅 2 行差异**（logo/url）；
  插入裸映射条目 → 自动补 `- ` 并归一到 6 空格缩进、落在分组内正确位置；
  删除 → 无连续空行；非法块 YAML → 400；只读用户 → 403；匿名 → 401；写入后生成 `.bak`。
- 前端 `pnpm run build` 通过，新组件无 Svelte 警告（`YamlEditor` 的 a11y 警告是改动前就存在的，留待第 5 批）。

### 0.5 第 4 / 5 批实际改动清单

**第 4 批：样式与主题**

| 文件 | 改动 |
| --- | --- |
| `stores.js` | `getThemeColors` → `getThemeTokens`，输出语义 token：`--bg-deep/-mid/-grad`、`--surface(-hover)`、`--border(-strong)`、`--ink(-muted/-soft)`、`--accent(-soft/-line/-ink)`；新增 `hexToRgba()`，不再用「色值 + `33`」拼字符串（那只对 6 位 hex 成立）；6 个主题的 4 个色值重新定位为「深色底 / 深色底 / 强调色 / 浅色文字」，并修正了 `arctic-frost` 两段底色相同、`midnight-galaxy` / `forest-canopy` 副色偏亮的问题 |
| `App.svelte` | 注入 token 并补 `:root` 兜底；`body` / `.no-results` / `.footer` 改用 token；`.theme-wrapper` 由三段渐变（右下角铺到浅色）收敛为两段深色渐变；`.services-container` 的列数改由 `--columns` 参与计算（`max(280px, 均分宽度)` + `auto-fit`），删掉会硬覆盖成 2 列的媒体查询；无标签时不渲染 sidebar；新增 `prefers-reduced-motion` |
| `Navbar` / `TagFilter` / `ServiceGroup` / `ServiceItem` / `Toolbar` / `Header` / `LoginModal` / `YamlEditor` | 所有 `var(--theme-primary…)` 前景色改用 `var(--accent…)`；`--theme-background` → `--ink`；弹窗改用固定深色玻璃面；权限徽标（rw/ro/no）改用固定语义色（绿/黄/红）而非主题背景色；`logo-fallback` 的「深色渐变 + 白字」改为中性面 + 强调色；`TagFilter` 移动端由整体隐藏改为横向滚动；`TagFilter` / `ServiceGroup` / `ServiceItem` 的选中态与 hover 用 `--accent-soft` / `--accent-line`；`#ffffff` 文本色统一为 `var(--ink)` |
| `index.html` / `main.js` | `lang="zh-CN"`；删除无用的注释与内联 style；Svelte 挂载到 `#app`（原来挂在 `document.body`，`#app` 是死元素）；`bundle.css` 移到 `<head>` |

**第 5 批：工程收尾**

| 项 | 改动 |
| --- | --- |
| 死代码 | 删除只有 `package service` 的 `internal/service/` |
| `internal/util.go` | `getContentType` 改为「显式表 + `mime.TypeByExtension` 兜底」，补上 `woff2/woff/ttf/otf/eot/webp/avif/map/txt/webmanifest`，并对文本类型带上 `charset=utf-8`；`downloadIconFile` 改为临时文件 + rename |
| 日志 | 新增 `Server.debugf`，静态资源的逐请求日志只在 `mode: debug` 下打印 |
| 图标缓存健壮性 | 下载失败不再返回 500，改为 302 跳转到 CDN 让浏览器兜底；下载中断也不会留下被当成有效缓存的半截文件 |
| `deploy/Dockerfile` | 重写为多阶段：`node:20-alpine` 构建前端 → `golang:1.25-alpine` 构建后端 → `alpine` 运行；修正了原来不存在的 `backend/` 目录与「前端无需构建」的错误注释 |
| `deploy/docker-compose.yml` | 端口统一 8090；`build.context: ..` + `dockerfile: deploy/Dockerfile`；挂载 `../pages`（**不加 `:ro`**，否则页面编辑无法保存）与 `../config.yaml:ro`；去掉已废弃的 `version` |
| `pages/*.yaml` | 38 个图标由 `cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/...` 改为 `icons-local/dashboard-icons/png/...`（走本地缓存）；按 homarr-labs 的实际名称修正 `pihole` → `pi-hole`、`gcp` → `google-cloud`；`stack-overflow` 在两个图标源里都不存在（改前就是 404），去掉 logo 以使用首字母兜底 |
| `README.md` / `project.md` | 功能列表补充单块编辑与权限体系；访问端口 8080 → 8090；API 表改为实际路由（`op=r` / `op=blocks` / `op=w` / `op=block`、`/api/login`、`/api/logout`，删掉并不存在的 `/api/health`、`/api/auth`）；主题补全为 6 个并说明语义 token；认证示例去掉不必要的 `/*:ro`、补上 `deny` |

**验证（第 4 / 5 批）**

- `go build` / `go vet` / `go test` 全绿，`gofmt -l internal/` 为空，`go test -race -count=2` 在 `golang:1.25` 容器内通过；前端 `pnpm run build` 通过且**零 Svelte 警告**（第 4 批顺带修掉了 `YamlEditor` 原有的 a11y 警告）。
- 端到端（release 模式）：4 个页面均正常解析（4/3/3/2 个分组）；图标缓存实测下载并落盘（`plex` / `pi-hole` / `google-cloud` 三个名字均成功），命中缓存后 `200 image/png`；`woff2` → `font/woff2`、`css`/`js`/`html` 均带正确类型与 charset；release 模式下静态请求日志行数为 **0**。
- 说明：**未**做「把所有 `rgba(255,255,255,x)` 表面色也换成 token」的全量替换 —— 这些是深色面上的半透明白，对当前 6 个主题都成立，全量替换只产生无意义的 diff。真正导致不可读的是「深色背景色当前景色」，已全部修掉。

---

## 0. 现状与结论速览

| 方向 | 结论 | 最严重的问题 |
| --- | --- | --- |
| 权限设计 | 模型能跑，但语义含糊 + 存在逻辑缺陷 | 已登录用户不回退匿名基线，导致「登录后反而 403」 |
| 页面编辑 | 可用但脆弱，且粒度太粗 | 只能整文件改原始 YAML；YAML 校验失败返回 HTTP 200 → **静默丢失用户修改** |
| 页面样式 | 能看，但主题色语义用反 | `--theme-primary`（深色背景色）被当作前景色 → **选中态/强调色不可读** |

---

## 1. 优先级总览

| 编号 | 问题 | 严重度 | 影响面 | 改动范围 |
| --- | --- | --- | --- | --- |
| P0-1 | YAML 校验失败返回 200，前端误判保存成功（静默丢数据） | 高（数据丢失） | 编辑功能 | `internal/handlers.go`、`YamlEditor.svelte` |
| P0-2 | 已登录用户不回退匿名基线 → 公开页面 403（降权） | 高（功能不可用） | 全部页面 | `internal/auth.go` |
| P0-3 | `PageDataManager.cacheMap` 无锁并发写 → race / panic | 高（进程崩溃） | 全部请求 | `internal/page.go` |
| P1-1 | 权限语义含糊：`!` 只对匿名生效、`:no` 静默失效、`*` 前缀会匹配 `/innerfoo` | 中高（安全/可预期性） | 权限配置 | `internal/config.go` |
| P1-2 | 前后端各写一份路径匹配算法，必然漂移 | 中 | 权限 + UI | `Toolbar.svelte`、`internal/*` |
| P1-3 | 编辑粒度太粗：只能整文件改原始 YAML，无脏检查 / 校验定位 | 中 | 编辑功能 | `YamlEditor.svelte`、新增 block API |
| P1-4 | 保存非原子（`os.WriteFile` 直写）+ 无备份 + 无冲突检测 | 中 | 编辑功能 | `internal/handlers.go` |
| P1-5 | 图标搜索插入远程直链，绕过 `icons-local` 本地缓存且预览失真 | 中 | 编辑功能 | `IconSearch.svelte` |
| P1-6 | 主题色语义用反 → 选中态不可读；白字硬编码，主题只换背景 | 中 | 全部页面 | `stores.js`、全部组件 `<style>` |
| P1-7 | 列表视图 `.url-popover` 绝对定位错乱，永久浮在行上 | 中 | 列表视图 | `ServiceItem.svelte` |
| P2-1 | 缺少页面管理（新建/重命名/删除），无法编辑 navs / page_defaults | 中 | 编辑功能 | 前后端 |
| P2-2 | 响应式列数覆盖 `columns` 配置（≤1400px 强制 2 列） | 低 | 卡片视图 | `App.svelte` |
| P2-3 | TagFilter 在 <768px 直接 `display: none`，手机无法筛选 | 低 | 移动端 | `TagFilter.svelte` |
| P2-4 | 认证周边：session 无清理、cookie 无 Secure、logout 可 GET、登录无限速 | 中低 | 认证 | `internal/auth.go`、`internal/server.go` |
| P2-5 | `getContentType` 缺 woff2/webp/ttf/map | 中低 | 静态资源 | `internal/util.go` |
| P2-6 | `internal/service/` 空壳死代码 | 低 | 代码卫生 | `internal/service/` |
| P2-7 | 路径遍历纵深防御缺失（`a/../../b`） | 中低 | 文件读写 | `internal/page.go`、`handlers.go` |
| P2-8 | deploy/Dockerfile 与 docker-compose 已失效 | 中 | 部署 | `deploy/*` |
| P2-9 | 文档漂移（README 旧 auth 格式、主题数量不一致、未使用字段） | 低 | 文档 | `README.md`、`project.md` |
| P2-10 | `pages/*.yaml` 图标直链旧仓库 `walkxcode`，缓存机制未被使用 | 中低 | 图标 | `pages/*.yaml` |
| P2-11 | 静态资源每请求 `log.Printf`，日志噪音 | 低 | 运维 | `internal/handlers.go` |

---

## 2. P0：正确性与安全（建议第一批落地）

### P0-1 修复「YAML 校验失败 → 静默丢失用户修改」

**现象 / 根因**

`internal/handlers.go::handlePagePost` 在校验失败分支用了 `sendJSON`：

```go
if err := yaml.Unmarshal([]byte(req.Content), &testConfig); err != nil {
    s.sendJSON(w, map[string]interface{}{          // ← 没有设置状态码
        "success": false,
        "error":   "YAML 格式错误: " + err.Error(),
    })
    return
}
```

`sendJSON` 不写状态码 → 响应是 **HTTP 200**，且被再次包一层成 `{success:true, data:{success:false, error:...}}`。

前端 `YamlEditor.svelte::handleSave` 只判断 `if (!response.ok)`：

```js
if (!response.ok) { const data = await response.json(); throw new Error(data.error || '保存失败'); }
dispatch('save-success'); handleClose();   // ← 200 时走到这里，弹窗关闭、页面重载
```

结果：用户写了非法 YAML → 界面提示成功并关闭 → 重新加载的是**旧内容**，用户的编辑彻底消失。

**改法**

1. 后端：把「YAML 校验失败」改为标准错误响应，使用 400：

   ```go
   if err := yaml.Unmarshal([]byte(req.Content), &testConfig); err != nil {
       s.sendError(w, "YAML 格式错误: "+err.Error(), http.StatusBadRequest)
       return
   }
   ```

   并检查是否还有其他 `sendJSON` 被误用于错误分支（`handlePagePost` 的写入成功分支保持 `sendJSON` 即可）。
2. 前端：双保险，成功判定改为「HTTP OK **且** `data.success === true`」：

   ```js
   const payload = await response.json().catch(() => null);
   const ok = response.ok && payload && payload.success;
   if (!ok) throw new Error((payload && payload.error) || '保存失败');
   ```
3. 保存失败时**不要关闭弹窗**，保留用户输入。

**测试**

- Go：新增 `internal/handlers_test.go`，用 `httptest` 发 POST `?op=w` + 非法 YAML，断言 `400` 且 body `success=false`。
- 手工：编辑页故意把缩进写坏 → 应停留在弹窗并显示错误，内容不丢。

**验收**：非法 YAML 时前端报错且弹窗不关闭；页面配置文件未被改写（可对文件做 mtime 断言）。

---

### P0-2 修复「已登录用户不回退匿名基线 → 公开页面 403」

**现象 / 根因**

`internal/auth.go::BasicAuthMiddleware` 对已登录用户**只查该用户自己的规则**，不命中就直接 403：

```go
if username != "" {
    authConfig, exists := s.config.MatchUserAuthConfig(username, reqPath)
    if !exists || authConfig.Permission == PermNO {
        s.sendError(w, "Forbidden", http.StatusForbidden)   // ← 没有回退到游客规则
        return
    }
    ...
}
```

后果示例：

- 配置 `admin:admin123@*:rw` + `user2:pass@/x:rw`，`user2` 登录后访问公开的 `/` → **403**，而游客却能正常看。
- 且这与 `FilterNavsByPermission` 的行为矛盾：导航栏会**显示**公开页面（因为它对所有人可见），点进去却 403。

**改法**：改为「用户规则命中则覆盖，未命中则回退匿名基线」的并集语义（详见 §3.1 的 `Resolve`）。核心：

```
perm = match(userRules, path)  若命中 → 用之（含 no）
       否则 → match(guestRules, path)
       都没有 → 拒绝（fail closed）
```

**测试**：`internal/perm_test.go` 增加回归用例 `TestResolve_LoggedInFallsBackToGuestRules`。

**验收**：任何用户登录后，凡是游客能访问的路径都能访问；`/inner-tools` 之类仍按配置受控。

---

### P0-3 修复 `PageDataManager` 并发写 map

**现象 / 根因**

`internal/page.go::GetPageConfig` 与 `ClearCache` 直接读写 `m.cacheMap`，没有同步：

```go
if m.cacheMap == nil { m.cacheMap = make(map[string]*PageConfig) }  // 并发下 map 重复初始化
if page, ok := m.cacheMap[name]; ok && !refresh { return page, nil }
...
m.cacheMap[name] = page      // ← 无锁写
```

HTTP handler 天然是多 goroutine 并发，debug 模式（`refresh=true`）每次都会走到写分支 → `fatal error: concurrent map writes`。

**改法**：给 `PageDataManager` 加 `sync.RWMutex`（读缓存用 `RLock`，写/删用 `Lock`），并把懒初始化挪到 `Init()` 里做一次性初始化。不要用全局变量裸 map。

**测试**：`go test -race ./internal/...` 增加 `TestGetPageConfig_Concurrent`，用 `sync.WaitGroup` 起 50 个 goroutine 并发读 + 交替 `refresh=true`。

**验收**：`go test -race` 无告警，压测下不再 panic。

---

## 3. P1：权限模型重构（详细设计）

### 3.1 统一入口 `Resolve`

现在权限判断散落在 4 个地方，语义各不相同：

| 位置 | 用途 | 语义 |
| --- | --- | --- |
| `Config.IsNeedAuth(path, isWrite)` | 匿名请求是否需要登录 | 只看游客规则 |
| `Config.MatchUserAuthConfig(user, path)` | 已登录用户是否有权 | 只看该用户规则 |
| `Config.MatchAuthConfig(path)` | **仅测试使用**，生产未调用 | 遍历所有用户，取首个 |
| `Toolbar.svelte::matchPermForPath` | 前端算能不能编辑 | JS 里重写了一遍 |

**建议**：收敛成单一函数，后端 handler / 中间件 / 导航过滤 / 前端展示全部基于它。

```go
// internal/perm.go
package internal

// Perm 权限常量：rw 读写 / ro 只读 / no 拒绝
type Perm string

const (
    PermRW Perm = "rw"
    PermRO Perm = "ro"
    PermNO Perm = "no"
)

// Access 一次鉴权的结果
type Access struct {
    Perm      Perm   // 最终权限
    Allowed   bool   // Perm != no
    CanWrite  bool   // Perm == rw
    Authed    bool   // 是否凭已登录身份通过
    MatchedBy string // 命中的规则（便于日志/排障），形如 "@:/*:ro"
}

// Resolve 统一鉴权入口，优先级见 §3.4。
// username == "" 表示匿名，只走匿名基线（且此时 `!` 认证墙生效）。
func (c *Config) Resolve(username, reqPath string, isWrite bool) Access {
    // 0) 硬拒绝：任何人都不能访问
    if p, ok := c.matchHardDeny(reqPath); ok {
        return Access{Perm: PermNO, MatchedBy: "deny:" + p}
    }

    // 1) 匿名基线：只取匿名规则里的「允许」部分；
    //    `!` 认证墙单独存 guestDeny，不混进允许规则
    base, matched := c.matchRules(c.guestAllowRules(), reqPath)

    if username == "" {
        // 匿名：先看认证墙
        if p, ok := c.matchRules(c.guestDenyRules(), reqPath); ok {
            return Access{Perm: PermNO, MatchedBy: "guest-deny:" + p}
        }
    } else if u, ok := c.userRules(username); ok {
        // 2) 已登录：用户规则优先（含显式 no）
        if p, hit := c.matchRules(u, reqPath); hit {
            base, matched = p, true
        }
        // 3) 未命中则沿用匿名基线（`!` 不参与，墙已越过）
    }

    if !matched {
        return Access{Perm: PermNO, MatchedBy: "<default-deny>"}
    }
    acc := Access{Perm: base, Allowed: base != PermNO, CanWrite: base == PermRW, Authed: username != ""}
    if isWrite && !acc.CanWrite {
        acc.Allowed = false
    }
    return acc
}
```

要点：

- **写操作必须登录**：当前匿名 `*:rw` 时中间件放行、handler 里又 `username == "" → 403`，两处结论不一致。改为 `Resolve` 里统一「`isWrite && username == "" → 拒绝`」，并在中间件用 `403` + 明确文案（不是 401，避免前端反复弹登录框）。
- 中间件只做一件事：`acc := Resolve(...)`；`!acc.Allowed` 时返回 401（未登录）或 403（已登录无权）。
- `handlePagePost` 里的 `username == "" → 403` 可以删掉，改为断言 `acc.CanWrite`。

### 3.2 统一并文档化路径匹配语义

现状歧义（`Config.pathMatch`）：

- 无 `*` 的 `/path` → 子树匹配（`/path` 与 `/path/**`）。
- 以 `*` 结尾的 `/inner*` → **纯字符串前缀**，因此会匹配 `/innerfoo`、`/inner-tools`、`/inner2`。
- `/tools/*` 与 `/tools*` 行为不同但看起来一样。
- 「排除规则被强制排到数组最前」是**隐式顺序契约**（`parseAuths` 里 `append(noPerms, normalPerms...)`），没有文档化，也没测试保证。

**建议**：

1. 解析阶段就归一化成显式结构，不再靠字符串后缀猜：

   ```go
   type pathRule struct {
       Pattern string       // 归一化后：/inner 或 /inner/** 或 /**
       Perm    Perm
       Kind    ruleKind     // exact | subtree | wildcard
       Raw     string       // 原始文本，仅用于日志
   }
   ```

2. 语义定为：
   - `*`、`/*`、`/**` → 匹配一切。
   - `/**` 结尾 → 子树匹配。
   - `/seg/*` → `*` 只在本段内通配（**新增**），跨段请用 `/**`。
   - 其余 → 精确匹配 + 子树（**保持现状兼容**）。
   - 存量 `/inner*` 在解析时归一化为 `/inner/**`，行为不变但语义清晰。
3. 规则优先级写进文档并用测试固化：`no` > 具体路径 > `/**`；同优先级「先声明者生效」。
4. 建议把「全局限黑名单」独立出来（见 3.4），不要再混在某个用户的规则串里。

### 3.3 修复 `:no` 静默失效

`parseAuths` 只识别 `:rw` / `:ro`：

```go
hasPerm := strings.HasSuffix(p, ":rw") || strings.HasSuffix(p, ":ro")
if !hasPerm { ...; p = p + ":ro" }     // ← "/a:no" 变成 "/a:no:ro"
```

`pathMatch` 用 `strings.Cut(pathWithPerm, ":")` 只切第一个 `:` → `pattern="/a"`, `perm="no:ro"`。`perm` 既不是 `rw` 也不是 `no`，于是：读请求放行、写请求 403。**用户在配置里写 `:no` 会得到一个静默错误的语义**。

**改法**：`parseAuths` 用 `strings.LastIndex(p, ":")` 取后缀，识别 `rw|ro|no` 三种；不识别的后缀按配置错误处理（`parseAuths` 返回 error，启动时 fail fast，而不是静默降级）。

### 3.4 `!` 的语义：推荐设计（已给结论）

看 `config.yaml`：

```yaml
- "@*,!/inner*"   # 注释：访客：所有路径可访问，无需认证，除了 /inner 开头的路径
```

`parseAuths` 把 `!/inner*` 归一化成 `/inner*:no`，且 `IsNeedAuth` **只遍历游客规则**。所以这条 `no` 的作用域是「匿名」——**已登录用户不受它约束**。

现状的毛病不是「行为错了」，而是**一个 `!` 被迫兼职表达两件事**，导致两种完全不同的意图无法区分：

- 意图 A：「`/inner*` 匿名看不了，登录后可以看」——当前行为是对的。
- 意图 B：「`/inner*` 只有 admin 能看」——当前行为是**提权漏洞**（任何带 `/*:ro` 的用户登录后即可访问）。

#### 推荐结论

**`!` 只表示「认证墙」（匿名不可访问，登录后按各自权限），不表示「全局拒绝」；「谁都别想访问」用新的顶层 `deny` 表达。**

拆成两个正交概念，各自有明确写法：

| 概念 | 写法 | 含义 | 作用范围 |
| --- | --- | --- | --- |
| 公开可读 | `@path:ro`、`@*` | 匿名即可访问 | 所有人的基线 |
| **认证墙** | `!path` | 匿名不可访问；登录后按各自权限 | **仅匿名** |
| 用户权限 | `user:pass@path:rw` | 该用户对该路径的权限 | 仅该用户 |
| **硬拒绝** | 顶层 `deny: [path]` | 任何人都不通过（临时下线页面用） | 所有人 |
| 默认 | 无规则命中 | 拒绝（fail closed） | — |

`Resolve` 的优先级（从高到低）：

1. `deny` 命中 → **无条件拒绝**（不再有「更具体的允许可以覆盖」这种绕脑子的例外）。
2. 已登录：用户规则命中 → 用之；用户规则里命中的 `no` → 拒绝。
3. 已登录：用户规则未命中 → 回退匿名基线（此步**忽略 `!` 规则**，因为人已经登录、墙已越过）。
4. 匿名：匿名基线里 `!` 命中 → 拒绝（返回 401，触发登录框）。
5. 都没命中 → 拒绝。

#### 为什么不让 `!` 表示「全局拒绝」

`config.yaml` 里有 `page_navs` 的 `InnerTools → /inner-tools`，`FilterNavsByPermission` 已经把它对匿名藏起来、对已登录用户显示。如果 `!` 变成「谁都别想访问」，这个导航项对任何用户都永远点不开 —— 与配置意图自相矛盾。所以「认证墙」才是它的正确定位。

#### 推荐配置写法

```yaml
auths:
  # 1) 用户规则：只写「额外获得的权限」，读权限由匿名基线兜底（见下）
  - admin:admin123@*:rw
  - user1:user123@/tools:rw

  # 2) 匿名基线，两种写法按「最小权限」程度二选一：

  # 写法 A（推荐）：显式白名单，未列出的路径天然需要登录 + 需要授权
  - "@/:ro"
  - "@/tools:ro"
  - "@/another:ro"

  # 写法 B（等价于现状）：全站公开只读，/inner* 立认证墙
  # - "@*,!/inner*/**"

# 3) 硬拒绝：任何人都不能访问（临时下线、或明确敏感路径）
deny:
  - /inner-tools
```

**这套设计解决的三件事**：

1. **`user1` 不再需要写 `/*:ro` 兜底**。当前配置要求每个用户都补一条宽泛读权限，否则登录后就 403（P0-2）；而一旦写了 `/*:ro`，它又等价于「给该用户开全站读」，等于把「只有特定人能看」的口子彻底放开 —— 这是个双输的陷阱。改成「登录后自动继承匿名基线」之后，用户规则里只写真正的额外授权即可。
2. **意图 A / 意图 B 有了各自的写法**，不再靠碰运气：
   - 要 A：用 `!inner*/**`（认证墙）。
   - 要 B：**不要**用 `@*`，改成写法 A 的白名单，**并且**不给其他用户配 `/inner*` → 只有 `admin@*:rw` 能进；或者在白名单基础上加 `deny: [/inner-tools]`（连 admin 也进不去）。
3. **`!` 的实现变简单**：`!` 规则只在「未登录」这条分支参与匹配，不需要再往 `PathPerms` 里塞 `:no` 和它的隐式排序契约（§3.2 的第 4 点随之消解）。

配套改动：

- `parseAuths` 把 `!path` 解析到独立的 `guestDeny` 列表，而不是混进 `PathPerms`；`deny` 解析到 `hardDeny` 列表。
- 文档与示例（`README.md`、`config.yaml` 注释）同步改写：明确「`!` = 需要登录」「`deny` = 任何人禁止」。
- 测试固化：`@*,!/inner*/**` 下，匿名 `/inner-tools` 必须 401；`user1`（无 `/inner*` 规则）登录后必须 200；`deny: [/inner-tools]` 下 `admin` 也必须 403。

> 这一节是本文档里唯一「改变配置模型」的部分，**已按此方案实施**（第 2 批，2026-09-25），落地清单见 §0.3。

### 3.5 后端下发 `can_write`，前端不再重算

`Toolbar.svelte` 里 `matchPermForPath` + `hasWritePermissionForRoute` 是 Go `pathMatch` 的 JS 复刻。任何一边改了另一边就悄悄错（例如 3.2 改了 `*` 语义，前端就失配）。

**改法**：

- 在 `PageDataResponse` 增加字段：

  ```go
  type PageDataResponse struct {
      *PageConfig
      Navs     []NavItem   `json:"navs"`
      UserInfo *LoginInfo  `json:"user_info,omitempty"`
      CanWrite bool        `json:"can_write"`   // 新增：当前用户对当前页面是否可写
  }
  ```

  `handlePageGet` 里用 `s.config.Resolve(username, path, true).CanWrite` 填充。
- 前端：`Toolbar` 用 `$pageConfig.can_write` 控制「编辑」按钮显隐；删除 `matchPermForPath` / `hasWritePermissionForRoute` 两个函数。
- 同理，「权限明细下拉」可以继续用 `user_info.permissions`（展示用），但**判断逻辑只信后端**。

### 3.6 认证周边加固

| 问题 | 位置 | 改法 |
| --- | --- | --- |
| session 过期项无清理，内存无界 | `Server.sessions` | 增加 `time.Ticker` 定期清理（或惰性清理 + 上限），进程退出前停止 ticker |
| cookie 无 `Secure`，`MaxAge` 缺失 | `LoginHandler` | `Secure: r.TLS != nil \|\| 配置开启`；`MaxAge: int(ttl.Seconds())` 与服务端 TTL 对齐 |
| `/api/logout` 允许 GET → 可被 CSRF 登出 | `LogoutHandler` | 只允许 POST |
| 写操作无 CSRF 防护 | `handlePagePost` | 校验 `Origin`/`Sec-Fetch-Site: same-origin`（低成本方案），或引入 CSRF token |
| 登录无限速 | `LoginHandler` | 简单 IP+账号维度的失败计数/退避（内存即可） |
| `MatchAuthConfig` 仅测试使用 | `config.go` | 重构后删除，避免测试与生产走不同代码路径 |
| `Server.fs` 字段未使用 | `server.go` | 删除 |

### 3.7 权限部分测试计划（`internal/perm_test.go`）

表驱动 + `httptest`，用例覆盖 README 中宣称的所有配置形态：

1. **匿名基线**：`@*`；`@*,!/inner*`；`@/public:ro,/api`。
2. **用户覆盖**：`user1:user123@/tools:rw,/*:ro` → `/tools` = rw、`/` = ro、写 `/` = deny。
3. **回退回归（P0-2）**：`user2:pass@/x:rw` 且存在 `@*` → `/` 必须 allowed（当前会 403）。
4. **`no` 生效**：用户规则 `@*,!/inner*` + `user1@/*:ro` → 断言 `/inner-tools` 的实际结果，并与文档声明的语义一致。
5. **`*` 边界**：`/inner*` 对 `/inner`、`/inner-tools`、`/innerfoo`、`/innerX/y` 的匹配（固化 3.2 的决定）。
6. **`:no` 显式配置**：`admin@/a:no,/b:rw` → `/a` 读写均 deny（当前是「读放行」的 bug）。
7. **写操作必须登录**：匿名 + `*:rw` → POST 必须拒绝。
8. **fail closed**：无任何规则的空配置 → 全部 deny。
9. `go test -race` 全绿。

---

## 4. P1：页面编辑

### 4.1 编辑粒度：从「整文件」改为「单块修改」

现状：唯一的编辑入口是 `YamlEditor.svelte`，把**整个页面的原始 YAML** 塞进一个裸 `<textarea>`。用户只想改一个站点的 URL，却必须先读懂整个文件结构、找到那一行、把缩进对齐 —— 出错概率高，而且一次误操作影响整页。

**已决定：不引入 CodeMirror，不做整文件编辑；改为「点开某一张卡片，在弹窗里只改这一个块」，表单 ↔ 源码可切换。** 整文件编辑降级为「高级/兜底」入口（见 4.1.3）。

#### 4.1.1 交互设计

1. 卡片 / 列表项 hover 时（且有 `rw` 权限）出现一个小「编辑」图标。
2. 点击 → 弹窗「编辑站点」：
   - **表单模式**（默认）：`name` / `url` / `logo` / `subtitle` / `tags` / `target` 六个字段；`logo` 旁挂「从图标库选」（复用 `IconSearch`）。
   - **源码模式**：只显示**这一个 item 的 YAML 片段**（含前导 `- ` 与缩进），语法错误就地提示。
   - 顶部 `表单 | 源码` 切换按钮。
   - 双向同步：
     - 表单 → 源码：切到源码时用 `js-yaml` 把表单对象 dump 成片段。
     - 源码 → 表单：切回表单时解析片段；**解析失败就留在源码模式**并提示错误，绝不用半成品覆盖表单。
   - 字段校验：`name` 必填；`url` 必填且为 `http(s)://` 或站内绝对路径；`target` 仅允许 `_blank` / `_self`；`tags` 按逗号/空格转数组。
3. 弹窗底部：`删除此卡片` / `取消` / `保存`。
4. 分组（service）本身同样可编辑：名称 + 图标；并带「在此分组内新增卡片」。
5. 权限：`$pageConfig.can_write`（见 §3.5）为 false 时，**不渲染任何编辑入口**。

#### 4.1.2 关键问题：只改一块，如何不破坏文件里的注释和格式

`pages/*.yaml` 里有中文注释。如果走「解析成对象 → 改字段 → 整个文件重新 dump」，注释、空行布局、字段顺序会**全部丢失** —— 每改一个卡片就把用户的注释洗掉，不可接受。

**方案：定位到源码行区间，做文本级替换，其余部分原样保留。**

- 后端用 `goccy/go-yaml` 的 `yaml.Node` 解析，其 `Token` 带 `Position.Line/Column`，据此算出每个 item / service 在原文中的**行区间**。
- 新增两个 API（`PageApiHandler` 需要从现在的 `op=r|w` 两分支改成按 `op` 分发的表驱动）：

```
GET  /api/page/{path}?op=blocks
     → { blocks: [
          { kind: "service", index: 0, name: "Media", icon: "fas fa-play-circle",
            line_range: [12, 34] },
          { kind: "item", service_index: 0, index: 0,
            value: { name: "Plex", url: "...", tags: ["app"], ... },
            line_range: [15, 21],                    // 仅该 item 的源码行范围
            yaml: "- name: \"Plex\"\n  url: \"...\"\n" }   // 该段原始文本，供源码模式编辑
       ] }

POST /api/page/{path}?op=block
     body: { kind: "item" | "service", service_index: 0, index: 2,
             action: "update" | "insert" | "delete",
             yaml: "<该块的 YAML 文本>" }
     → 后端按 line_range 做原文替换 / 插入 / 删除，再原子写回（见 4.2）
```

- `update` / `insert` / `delete` 三个动作 + `kind` 两个层级，即可覆盖「改卡片 / 加卡片 / 删卡片」以及「改分组 / 加分组 / 删分组」，**共用同一套实现**。
- 收益：注释、空行、字段顺序、引号风格全部保留；改动面精确到几行，diff 干净，便于将来接 git 版本管理。
- 兜底：若行区间定位失败（文件被手工改乱、缩进异常），返回 `409` 并提示「请改用整文件编辑」。

#### 4.1.3 保留的整文件编辑（高级入口）

保留现有 `YamlEditor.svelte`，但移到「高级」入口（工具栏 `更多 → 编辑原始 YAML`），它今天的这些缺陷仍需修复：

| 问题 | 改法 |
| --- | --- |
| 无脏检查：Esc / 点遮罩 / 保存中都能直接关闭 → 丢改动 | 维护 `dirty`（`value !== original`）；关闭前 `if (dirty && !confirm('有未保存的修改，确定关闭？')) return`；`saving` 中禁止关闭（参照 `LoginModal` 的 `loading` 守卫） |
| Esc 直接关闭 | 同上；另外 Esc 监听移到弹窗内（全局 `svelte:window` 会干扰页面其他 Esc 行为） |
| 无 Ctrl/Cmd+S | `svelte:window on:keydown` 里判断 `(e.ctrlKey \|\| e.metaKey) && e.key === 's'` → `handleSave()` 并 `preventDefault` |
| 校验错误只有一句后端字符串、无行号 | 后端改用 `yaml.FormatError(err, true, true)` 返回**带行号与源码片段**的错误；前端按行号滚动定位并选中 |
| 无「格式化」入口 | 加「格式化」按钮：解析后重新 dump（注意：格式化会丢注释，需二次确认） |
| `document.querySelector('.yaml-editor')` 脆弱 | 改为 `bind:this={textareaEl}` |
| `dispatch('save-success')` + `handleClose()` 双重关闭 | 只保留 `save-success`，由父组件统一关闭 |
| 无 beforeunload 保护 | 弹窗打开且 dirty 时注册 `beforeunload`，关闭时注销 |

> 轻量说明：源码模式只编辑一个小片段，用 `<textarea>` + 简单行号 gutter 足够；不引入 CodeMirror，保持「极轻量」定位。若后续仍想要高亮，可再评估轻量方案，不作为本批范围。


### 4.2 保存的健壮性（`internal/handlers.go`）

1. **原子写入**：`os.WriteFile` 直写，进程/机器中途挂掉会留下半截 YAML（下次启动直接解析失败）。

   ```go
   func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
       dir := filepath.Dir(path)
       tmp, err := os.CreateTemp(dir, ".hpx-*.tmp")
       if err != nil { return err }
       tmpName := tmp.Name()
       defer os.Remove(tmpName)          // 成功时已被 rename，Remove 失败无害
       if _, err = tmp.Write(data); err != nil { tmp.Close(); return err }
       if err = tmp.Sync(); err != nil { tmp.Close(); return err }
       if err = tmp.Close(); err != nil { return err }
       if err = os.Chmod(tmpName, perm); err != nil { return err }
       return os.Rename(tmpName, path)
   }
   ```

2. **备份**：写入前把原文件复制为 `{name}.yaml.bak`（或按时间戳保留最近 N 份），给用户一个后悔的机会。
3. **并发冲突**：保存请求带上读取时的 `mtime` 或内容 hash，服务端比对不一致则返回 `409`，前端提示「配置已被他人修改，请刷新」。单人使用时可先跳过。
4. **写入前做路径安全校验**（见 P2-7）。
5. **权限**：`acc := Resolve(username, path, true); if !acc.CanWrite { 403 }`。

### 4.3 图标插入走本地缓存（`IconSearch.svelte`）

现状插入的是远程直链：

```js
// dashboard-icons
return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons@master/png/${icon.name}.png`;
```

这有两个问题：

1. **绕过了 `icons-local` 缓存机制**（`internal/handlers.go::GetIconLocalHandler` + `config.yaml::icons_cdn` 白做了）。
2. `@master` 是浮动引用；且 CSS 里 `.icon-image { filter: brightness(0) invert(1) }` 把所有彩色 logo 变成白色剪影，**预览与最终效果不一致**（实际页面上是彩色的）。

**改法**：

- 插入值改为本地缓存形式，与 `config.yaml` 的 `icons_cdn` key 对应：

  ```
  dashboard → icons-local/dashboard-icons/png/{name}.png
  selfhst   → icons-local/selfhst-icons/png/{name}.png
  ```

  这样首次访问时后端会自动下载并缓存到 `{frontend_dir}/icons-local/...`。
- 图标源列表不要在前端硬编码（现在 `ICON_SOURCES` 直接写死 jsdelivr 地址），应由后端把 `icons_cdn` 的 key 列表随页面数据下发，前端只负责拼路径。
- 去掉 `brightness(0) invert(1)`，保留原色；若深色背景下白色图标可读性差，则给卡片加浅色底而不是篡改图标。
- 元数据（`metadata.json` / `index.json`）目前由**浏览器**直接拉 jsdelivr，可考虑后端代理 + 缓存，避免“后端已缓存图标、前端仍强依赖外网元数据”的不一致；至少要把 `@master`/`@main` 固定到具体版本或做失败降级。

### 4.4 页面管理能力（P2，但属于「编辑没做好」的一部分）

现状只能编辑「已存在的页面」，没有任何管理入口（卡片 / 分组级的增删改已由 §4.1.2 的 `op=block` 覆盖，这里补的是**页面级与配置级**）：

- 不支持新建页面（`pages/{name}.yaml`），只能手动在磁盘上放文件。
- 不支持重命名 / 删除。
- 不能编辑 `config.yaml` 里的 `page_navs` / `page_defaults`，而导航栏恰恰由它们驱动 —— 用户会发现「我改了页面但导航没变」。
- 没有「当前页面列表」视图，编辑时只能靠 URL 猜。

**建议**（新增 API + 一个轻量的页面管理面板）：

```
GET    /api/pages            # 列出 pages_dir 下所有页面（name/title/mtime），按权限过滤
POST   /api/pages            # 新建页面 {name, template}
POST   /api/pages/rename     # {from, to}
DELETE /api/pages            # {name}
GET/POST /api/config         # 读写 config.yaml（仅 admin）
```

- 全部走 `Resolve` 鉴权；页面级操作要求目标路径 `rw`；`/api/config` 要求全局 `rw`/admin。
- 新建页面的 name 必须校验：`^[a-zA-Z0-9_-]+$`，且解析后的路径必须落在 `pages_dir` 内。
- 前端：编辑器头部加「页面选择器 + 新建」；`config.yaml` 编辑单独一个入口（与页面对话框区分，避免误操作）。

### 4.5 不要返回服务器绝对路径

`getPageRawContent` 返回：

```go
s.sendJSON(w, map[string]string{
    "content": string(content),
    "path":    pageConfig.Pagefile,   // 例如 D:\work\...\pages\home.yaml
})
```

没必要把服务器文件系统路径暴露给浏览器。前端未使用该字段，直接删除。

---

## 5. P1/P2：样式与主题

### 5.1 根因：主题色语义用反了

`stores.js::getThemeColors` 把 4 个颜色映射成 `primary/secondary/accent/background`，然后在 `App.svelte` 灌成 CSS 变量。问题是：

- 6 个主题的第 0 个颜色都是**深色背景色**（`#1a2332`、`#2d4a2b`、`#1e1e1e`…）。
- 但 `--theme-primary` 被当成**前景/强调色**到处用，于是「深色字压深色底」。

具体出错点（全部需要改）：

| 文件 | 选择器 | 问题 |
| --- | --- | --- |
| `Navbar.svelte` | `.nav-item.active { background: var(--theme-primary-rgba); color: var(--theme-primary) }` | 深字压深底，**当前页导航几乎不可读** |
| `TagFilter.svelte` | `.tag-btn.active { color: var(--theme-primary) }` | 同上 |
| `ServiceGroup.svelte` | `.group-header i { color: var(--theme-primary) }` | 分组图标看不清 |
| `ServiceItem.svelte` | `.item-title:hover { color: var(--theme-primary) }`、`.tag { color: var(--theme-primary) }` | hover 后标题变暗、标签不可读 |
| `Toolbar.svelte` | `.edit-btn` 边框/文字用 `--theme-primary` | 按钮看不清 |
| `Header.svelte` | `.btn-auth.login { border-color / color: var(--theme-primary) }` | 登录按钮看不清 |

**改法：拆出语义变量**，主题只提供色值映射，组件只用语义 token。

```js
// stores.js
export function getThemeTokens(themeId) {
  const t = themes.find(x => x.id === themeId) || themes[0];
  const [bgDeep, bgMid, accent, ink] = t.colors;
  return {
    '--bg-deep':      bgDeep,                 // 页面底色 / 深色面
    '--bg-mid':       bgMid,                  // 渐变中段 / 次级面
    '--bg-grad':      `linear-gradient(160deg, ${bgDeep} 0%, ${bgMid} 100%)`,
    '--surface':      'rgba(255,255,255,0.06)',
    '--surface-hover':'rgba(255,255,255,0.12)',
    '--border':       'rgba(255,255,255,0.12)',
    '--ink':          '#ffffff',              // 主文字（始终用白，保证与深底对比）
    '--ink-muted':    'rgba(255,255,255,0.62)',
    '--accent':       accent,                 // ★ 强调色 = colors[2]，浅色，可读
    '--accent-soft':  hexToRgba(accent, 0.16),// 选中态底色
    '--accent-ink':   bgDeep,                 // ★ 压在 accent 上的文字色
  };
}
```

所有 `color: var(--theme-primary)` 一律改为 `var(--accent)`；`background: var(--theme-primary-rgba)` 改为 `var(--accent-soft)`；`background: var(--theme-primary)` 改为 `var(--accent)` 并配 `color: var(--accent-ink)`。

> 验证：6 个主题里 `colors[2]` 分别是 `#a8dadc / #00ffff / #d3d3d3 / #a490c2 / #a4ac86 / #c0c0c0`，在各自的深色底上对比度都 ≥ 4.5:1，可安全作为前景强调色。

### 5.2 修掉 `色值 + "33"` 的字符串拼接

```js
'--theme-primary-rgba': `${themeColors.primary}33`,   // 只对 6 位 hex 成立
```

一旦主题色写成 `#abc`、`rgb(...)`、`hsl(...)` 就会产出非法 CSS，整条声明被丢弃（静默失效）。改用 `hexToRgba(hex, alpha)` 显式转换，或直接用 `color-mix(in srgb, var(--accent) 16%, transparent)`（现代浏览器支持）。

### 5.3 消除硬编码白色

组件里遍布 `color: #ffffff` / `rgba(255,255,255,.6)`，导致主题实际只换了背景渐变，文字色纹丝不动。浅色主题（`arctic-frost` 的 `#4a6fa5 → #c0c0c0`）会出现白字压浅灰。

**改法**：全部换成 `var(--ink)` / `var(--ink-muted)` / `var(--border)` / `var(--surface)`。

### 5.4 收敛多段渐变

`App.svelte`：

```css
.theme-wrapper { background: linear-gradient(135deg, var(--theme-primary), var(--theme-secondary), var(--theme-accent)); }
```

三段渐变让右下角变成 `--theme-accent`（浅青/浅灰），白字对比度随位置剧烈变化，footer 区域尤其糊。

**改法**：只留两段深色渐变（`--bg-deep → --bg-mid`），需要氛围感时用低透明度 `radial-gradient` 叠加，而不是把浅色铺满。

### 5.5 修列表视图 `.url-popover`（明显视觉 bug）

`ServiceItem.svelte`：

```css
.url-popover {
  position: absolute;
  /* top: 100%; */      /* ← 被注释掉 */
  right: 0;
  /* margin-top: 8px; */ /* ← 被注释掉 */
  ...
}
```

`.list-actions` 是 `position: relative`，所以这个「弹层」**永久停留在行内**并压住标题/副标题。而且 HTML 里没有任何 hover 显示/隐藏的逻辑。

**改法**：二选一

- 简单：改成 flex 内联元素（去掉 `position: absolute`），URL 常态化显示在行尾，超长省略。
- 保留 popover 交互：加 `hover`/`focus` 状态控制显隐，补回 `top: calc(100% + 8px)`，并处理溢出（`overflow: visible` 的祖先链）。

### 5.6 响应式列数与视图语义

- `App.svelte` 的 `.services-container` 用 `repeat(var(--columns,3), minmax(280px,1fr))`，但 `@media (max-width: 1400px)` 直接强制 2 列 → 1366 笔记本上 `columns: "4"` 失效。
  **改法**：用 `repeat(auto-fill, minmax(280px, 1fr))` 单一规则，或按 `min(columns, floor(width/threshold))` 计算；不要用媒体查询硬覆盖。
- `cards` / `list` 差异极小，且 `cards` 实际是「分组当卡片」。若想对齐 Homer 的直觉（每个服务项一张卡片），需要重做 `ServiceGroup` 的布局分支——属于体验增强，建议单独立项并配 `frontend-design` skill 做视觉稿。

### 5.7 移动端与可访问性

| 问题 | 改法 |
| --- | --- |
| `TagFilter` <768px `display: none` → 手机完全无法按标签筛选 | 改为横向滚动条（`overflow-x: auto; flex-direction: row`），保留功能 |
| sidebar 固定 200px，无标签时仍占位 | 无标签时不渲染 `aside`；或折叠成图标条 |
| `html lang="en"` 但界面为中文 | 改 `lang="zh-CN"` |
| `index.html` 里的 `<div id="app">` 是死元素（Svelte 挂载到 `document.body`） | 删除，并改为挂载到 `#app`（更标准，避免与 body 直接耦合） |
| 弹窗无焦点陷阱/自动聚焦 | `YamlEditor` / `LoginModal` 加 `autofocus` + 简单的 Tab 循环 + 关闭后归还焦点 |
| 图标按钮只靠 `title` | 补 `aria-label` |
| `backdrop-filter` 多处使用 | 保留但加 `@supports` 降级；`prefers-reduced-motion` 下关闭过渡 |

---

## 6. P2：代码与工程

| 编号 | 问题 | 改法 |
| --- | --- | --- |
| P2-6 | `internal/service/{page_service.go,perm_service.go}` 只有 `package service`，是空壳 | 删除，或按 §3.1 真正落地为 `perm.go` + `page_store.go`。不要留占位包 |
| P2-5 | `util.go::getContentType` 缺 `.woff2/.woff/.ttf/.webp/.map/.txt` | 补全；或直接用 `mime.TypeByExtension` + 兜底。**注意 selfhst 的图标就是 `.webp`**，现在以 `application/octet-stream` 返回 |
| P2-7 | 路径穿越纵深防御缺失 | `getFilename` 用 `TrimLeft(name, "/.")` 只处理开头；`a/../../b` 经 `filepath.Join` 会逃出 `pages_dir`。新增 `resolvePageFile(name)`：`filepath.Abs` 后校验 `filepath.Rel(pagesDir, resolved)` 不以 `..` 开头，否则报错。（Go 的 `ServeMux` 会先清洗 `..`，属纵深防御） |
| P2-11 | 静态资源每个请求 `log.Printf` | 降为 debug 级或只在 `mode: debug` 下打印；去掉动态资源的逐请求日志 |
| P2-8 | **`deploy/Dockerfile` 已失效**：`COPY backend/ ./backend/`、`RUN cd backend && go mod tidy`，但代码里没有 `backend/` 目录（只有根目录 `main.go` + `internal/`）；且注释写「Frontend is static, no build needed」但实际需要 `pnpm run build` | 重写 Dockerfile：多阶段（node 构建前端 → golang 构建后端 → alpine 运行），`COPY . .`、`COPY --from=frontend-builder /app/frontend/build ./frontend/build` |
| P2-8b | `deploy/docker-compose.yml`：暴露 `8080:8080` 但服务默认 `8090`；挂载 `./pages` 与 `./config`（后者不存在，配置是 `config.yaml`）；`build: .` 但 Dockerfile 在 `deploy/` 下，且依赖仓库根的 `pages/`、`config.yaml` | 统一端口为 8090；挂载 `../pages` 与 `../config.yaml`；`build.context: ..` + `dockerfile: deploy/Dockerfile` |
| P2-9 | 文档漂移 | 见下 |
| P2-10 | `pages/*.yaml` 图标全部直链 `cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/...`（旧仓库），而 `config.yaml` 里映射的是 `homarr-labs/dashboard-icons` → **缓存机制完全没被使用**，且旧仓库一旦失效全站图标挂 | 全部改为 `icons-local/dashboard-icons/png/xxx.png`，让后端缓存接管 |
| — | `connectivity`、`Item.Method/Headers/Type/Keywords` 解析了但无任何逻辑 | 要么实现（`connectivity` 已在 todo.md 里），要么从结构体/文档里移除，避免误导 |

**文档同步清单（P2-9）**

- `README.md` 的「认证配置」还在写旧的 `auth: { enabled, username, password }`，与现在的 `auths` 字符串格式不符。
- `README.md` 列出的 `/api/health`、`/api/auth`、`/api/logout` 与 `main.go` 实际注册的路由不一致（实际是 `/api/login`、`/api/logout`，没有 `/api/auth` 和 `/api/health` 路由）。
- 主题数量不一致：`project.md` 一处写「6 themes available」，紧接着只列了 5 个；`stores.js` 实际有 6 个（含 `tech-innovation`）。
- `project.md` 的 API 表格未包含 `?op=r` / `?op=w` 参数说明。
- `README.md` 说访问 8080，实际默认 8090。

---

## 7. 建议实施顺序

**第 1 批（P0，小改动、高收益、低风险）**

1. P0-1 YAML 错误返回 400（`handlers.go` + `YamlEditor.svelte`），补 `handlers_test.go`。
2. P0-3 `PageDataManager` 加锁，补 `-race` 并发测试。
3. P0-2 权限回退：先只加「用户未命中则回退匿名基线」，暂不动匹配语义，补回归测试。

> 这一批不动配置格式、不动 UI 结构，可以快速验证并发布。

**第 2 批（P1，权限模型收敛）**

4. 引入 `internal/perm.go` 的 `Resolve` + `Perm`，中间件/handler/导航过滤全部切过去（3.1）。
5. 归一化路径规则 + 支持 `:no` + 明确 `!` 语义（3.2 / 3.3 / 3.4）。
6. `PageDataResponse.CanWrite` 下发，删除前端匹配逻辑（3.5）。
7. 删除 `MatchAuthConfig`，测试切到 `Resolve`（消除假信心）。
8. 认证周边加固（3.6）。

**第 3 批（P1，编辑体验）**

9. 原子写入 + 备份 + 路径安全校验（4.2 / P2-7）——单块编辑依赖它，必须先做。
10. 单块编辑：`op=blocks` / `op=block` 行区间替换 API（4.1.2）+ 卡片编辑弹窗（表单 ↔ 源码，4.1.1）。
11. 整文件编辑兜底入口的修复：脏检查 / Ctrl+S / 带行号的错误（4.1.3）。
12. 图标插入改走 `icons-local`（4.3）。

**第 4 批（P1/P2，样式）**

13. 语义色 token 重构 + 全部组件替换（5.1~5.4）。这一步建议独立 PR，便于对照截图验收。
14. 修 `.url-popover`、响应式列数、移动端标签栏（5.5~5.7）。

**第 5 批（P2，工程收尾）**

15. 页面管理 API + 面板（4.4）。
16. `internal/service` 删除、`getContentType` 补全、日志降噪（P2-5/6/11）。
17. deploy 修复（P2-8）、文档同步（P2-9）、`pages/*.yaml` 图标改本地（P2-10）。

---

## 8. 验收方式（每批通用）

```bash
# 后端
go build ./...
go vet ./...
go test -race ./internal/...

# 前端
cd frontend && pnpm install && pnpm run build

# 手工回归清单
# 1. 匿名访问 /            → 正常显示
# 2. 匿名访问 /inner-tools → 弹登录框（不重复弹）
# 3. 登录受限用户 → 公开页可看；/tools 可编辑；/ 无「编辑」按钮
# 4. 编辑页写入非法 YAML   → 报错、弹窗不关、文件未被改写
# 5. 编辑页写入合法 YAML   → 保存成功、页面刷新为新内容
# 6. 列表视图 URL 显示正常，不遮挡标题
# 7. 6 个主题切换 → 选中态/标签/图标均可读（对比度检查）
# 8. 手机宽度 → 标签栏可横向滚动筛选
# 9. 卡片编辑：表单改 URL 保存 → 卡片立即更新；
#    且 pages/home.yaml 的注释与其余行未被改动（git diff 只有那几行）
# 10. 卡片编辑：表单/源码来回切换保持一致；源码写坏时留在源码模式并报错
# 11. 权限模型（若已实施 §3.4）：匿名访问 /inner-tools → 401；登录后按配置放行
```

---

## 9. 待确认 / 暂不做

- ~~**`!` 语义（§3.4）**~~：**已确认并实施** —— `!` = 「认证墙」（仅匿名不可访问），「谁都别想访问」用顶层 `deny`。落地清单见 §0.3。
- ~~编辑器引入 CodeMirror~~：**已决定不引入**，改为单块编辑（表单 ↔ 源码），见 §4.1（已实施，见 §0.4）。
- ~~`op=block` 的行区间定位~~：**已实施**，见 §0.4。缩进异常 / 手工改乱的文件定位失败时返回明确错误并引导使用整文件编辑。
- **是否重做 `cards` 视图**为「每个服务项一张卡片」（Homer 风格）：仍未做 —— 目前 `cards` 是「分组当卡片」。属视觉重塑，建议单独出稿。
- **多人协作/冲突检测（4.2 第 3 点）**：仍延后 —— 当前定位是个人仪表盘。
- **`AuthEnabled()` / `ParsedAuths()` / `AuthConfig.IsValid()`**：在改动前就没有调用方（属既有死代码），本次未删，避免扩大 diff；如需要可在后续清理。
- **`pages/*.yaml` 的 `connectivity`、`Item.Method/Headers/Type/Keywords`**：解析了但无逻辑，同上未动。
