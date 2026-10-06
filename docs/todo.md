# TODO

- [x] 支持按空格分割多个关键词，使用 AND 关系搜索
- [x] 优化增强：加载数据后收集所有的 tags信息，固定到 services-container 左外侧
  - 显示每个tag 的 item 数量
  - 点击指定 tag 后页面只显示对应的 items
- [x] 增强：如果在浏览器输入了 query 参数 `refresh=true` 需要将此参数带给后端 API `/api/page/*` 刷新缓存数据
- [x] 优化：dashboard-icons 图标缓存本地，避免远程加载。当 icon 配置为 `icons-local/dashboard-icons/png/plex.png` 时，先从缓存中查找，若不存在则从远程加载并缓存到本地目录。
  - 必须以 `icons-local/` 开头才会触发此逻辑
  - 缓存目录结构：`{frontend_dir}/icons-local/dashboard-icons/png/`
    （v0.3.0 起缓存目录改为可配置的 `icons_dir`，默认 `<配置文件目录>/icons-cache`）
  - eg: https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/plex.png -> `{frontend_dir}/icons-local/dashboard-icons/png/plex.png`

- [x] 增强：支持在线编辑配置文件，如果有权限，页面显示编辑按钮。打开弹窗，允许用户编辑yaml配置内容。
  - 编辑后，页面会自动刷新，显示最新配置
- [x] basic 认证改为了简单的普通登录处理

- [ ] 增强：支持简单的状态检测，用于判断服务是否可用。仅存储在内存中，不持久化。
  - 新增 `connectivity` 字段，用于配置状态检测
  - 新增 `mode` 字段，可选值为 `ping` 或 `http`
  - 新增 `check_interval` 字段，用于配置检测间隔（单位：毫秒）
  - 新增 `url` 字段，用于配置检测 URL（仅在 `mode` 为 `http` 时生效）

## 离线 / 内网单文件部署（v0.3.0）

- [x] 前端构建产物用 `go:embed` 内嵌进二进制：
  - `frontend_dir` 里有 `index.html` 时优先用目录，否则用内嵌资源
  - `-tags embedfrontend` 内嵌 `frontend/build`（Makefile 的构建目标都带该 tag，且先构建前端）；未带 tag 时内嵌 `frontend/placeholder` 占位页，保证 `go build/test` 在没有前端产物的机器上也能过
- [x] 图标缓存目录可配置：新增 `icons_dir`，默认「配置文件所在目录/icons-cache」
  （内嵌资源只读，缓存不能再写进 `frontend_dir`）
- [x] 新增 `icons_remote`（默认 `true`）：设为 `false` 时完全不访问 CDN，未缓存的图标直接 404
- [x] 图标下载失败进 10 分钟失败缓存，期间不再重试（离线时不再每个请求都等满 5s 超时）
- [x] `IconSearch` 拉取 CDN 元数据失败/超时（8s）时给出可操作提示，不阻塞编辑表单；
  后端 `icons_remote: false` 时前端不再请求元数据（`/api/page` 响应新增 `icons_remote`）
- [x] 发布包不再需要附带 `frontend/build`；`make build-linux` 产物在只有
  「二进制 + config.yaml + pages」的目录里可直接启动并提供前端

## 基于配置实现简单的 basic 认证

- [x] 增强：基于配置实现简单的 basic 认证支持
  - 权限按页面配置，有页面的权限就有对应 api 的权限（api 访问去除 api/page 前缀后检查）
  - 新增 `auths` 字段，用于配置认证规则
  - 每个规则格式为 `{username}:{password}@{path:perm},{path2:perm2}`
  - `@*` 表示所有路径可读
  - `:rw` 表示读写权限
  - `:ro` 表示只读权限(默认)
  - 路径可以使用通配符 `*` 表示任意路径，不带 `*` 表示精确匹配
  - 路径可以使用前缀 `!` 表示排除路径
