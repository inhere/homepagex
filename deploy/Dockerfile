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

COPY main.go ./
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o homepagex .

# ---------- 3. 运行 ----------
FROM alpine:latest

WORKDIR /app

# 图标缓存与远程下载需要 CA 证书
RUN apk --no-cache add ca-certificates

COPY --from=backend-builder /app/homepagex ./
COPY --from=frontend-builder /app/build ./frontend/build
COPY config.yaml ./
COPY pages/ ./pages/

EXPOSE 8090

CMD ["./homepagex"]
