# HomePageX 镜像：前端（Svelte/Rollup） -> 后端（Go） -> 运行镜像
#
# 注意：构建上下文是仓库根目录（见 docker-compose.yml 的 build.context: ..），
# 因此这里用 frontend/ 、internal/ 这类相对仓库根的路径。

# ---------- 1. 构建前端 ----------
FROM node:20-alpine AS frontend-builder

WORKDIR /app
RUN corepack enable

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm run build

# ---------- 2. 构建后端 ----------
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
# internal 依赖 frontend 包（内嵌资源入口），必须一起拷进构建阶段
COPY frontend/*.go ./frontend/
# 内嵌前端产物：与 Makefile 一致，用 -tags embedfrontend 读取 frontend/build
COPY --from=frontend-builder /app/build/ ./frontend/build/

ARG VERSION=0.3.0
RUN CGO_ENABLED=0 go build -trimpath -tags embedfrontend -ldflags "-s -w -X main.Version=${VERSION}" -o homepagex ./cmd/homepagex

# ---------- 3. 运行 ----------
FROM alpine:latest

WORKDIR /app

# 图标缓存与远程下载需要 CA 证书
RUN apk --no-cache add ca-certificates

# 前端资源已经内嵌进二进制，运行镜像只需要二进制 + 配置 + 可写的 pages
COPY --from=backend-builder /app/homepagex ./
COPY config.yaml ./
COPY pages/ ./pages/

EXPOSE 8090

# 全局选项要写在子命令前面（-c 是应用级选项）；容器里配置与页面都在 /app 下
CMD ["./homepagex", "-c", "/app/config.yaml", "serve"]
