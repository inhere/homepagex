# homepagex — Makefile

APP      := homepagex
MAIN_DIR := ./cmd/homepagex
GOEXE   = $(shell go env GOEXE)
GOPATH  = $(shell go env GOPATH)
BINARY  := $(APP)$(GOEXE)

# Build metadata
BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S)
GIT_HASH  := $(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "unknown")
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev-$(GIT_HASH)")

LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.GitCommit=$(GIT_HASH) \
	-X 'main.BuildDate=$(BUILD_TIME)'

# upx 压缩：未安装时自动跳过（ubuntu-latest 默认没有 upx，硬依赖会让 CI 直接失败）
UPX := $(shell command -v upx 2>/dev/null)
compress = $(if $(UPX),$(UPX) -6 --no-progress $(1),echo "   (skip upx: not installed)")

DIST_DIR := dist
RELEASE_DIR := release
WEB_DIR := frontend
# 发布包除二进制外还需要带上这些：homepagex 是从磁盘读取 frontend_dir 与 pages_dir 的，
# 光有二进制跑不起来。
RELEASE_FILES := config.yaml pages frontend/build
# 注：值会经 `echo "description: $(DESCRIPTION)"` 写入 latest.yaml，避免 `;`/`:`/引号等
# shell/YAML 元字符（否则 recipe 展开后会被截断成多条命令）。
DESCRIPTION := lightweight Homer-like dashboard homepage (Go + Svelte).

.PHONY: all build web-deps web-dist install run clean clean-dist help latest build-all dump-info latest-yaml release \
	build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows

## all: build (default)
all: build

## build: build frontend then Go binary (current platform)
build: web-dist
	@mkdir -p $(DIST_DIR)
	@echo "🐹 Building $(APP) $(VERSION) @ $(GIT_HASH)..."
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY) $(MAIN_DIR)
	@$(call compress,$(DIST_DIR)/$(BINARY))
	@echo "✅ Binary: $(DIST_DIR)/$(BINARY)"

## web-deps: install frontend dependencies (pnpm)
web-deps:
	pnpm --dir $(WEB_DIR) install --frozen-lockfile

## web-dist: build frontend assets into $(WEB_DIR)/build
web-dist: web-deps
	pnpm --dir $(WEB_DIR) run build

## install: install Go binary to $GOPATH/bin
install: build
	@cp $(DIST_DIR)/$(BINARY) $(GOPATH)/bin/$(BINARY)
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY)"

## run: build and run with current directory (needs config.yaml + pages + frontend/build)
run: build
	./$(DIST_DIR)/$(BINARY)

# ─── Cross Compilation ────────────────────────────────────────────────────────

## build-all: cross-compile for all platforms
build-all: clean-dist dump-info web-dist build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows latest-yaml
	ls -lh $(DIST_DIR)

## dump-info: dump build info
dump-info:
	@echo "Build Info:"
	@echo "  VERSION: $(VERSION)"
	@echo "  GIT_HASH: $(GIT_HASH)"
	@echo "  BUILD_TIME: $(BUILD_TIME)"

## latest-yaml: generate latest.yaml release metadata
latest-yaml:
	@mkdir -p $(DIST_DIR)
	@{ \
		echo "name: $(APP)"; \
		echo "version: $(VERSION)"; \
		echo "released_at: $(BUILD_TIME)"; \
		echo "description: $(DESCRIPTION)"; \
	} > $(DIST_DIR)/latest.yaml
	@echo "   → $(DIST_DIR)/latest.yaml"

## build-linux: compile for Linux amd64
build-linux: web-dist
	@echo "🐧 linux/amd64..."
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-amd64 $(MAIN_DIR)
	@$(call compress,$(DIST_DIR)/$(APP)-linux-amd64)
	@echo "   → $(DIST_DIR)/$(APP)-linux-amd64"

## build-linux-arm64: compile for Linux arm64
build-linux-arm64: web-dist
	@echo "🐧 linux/arm64..."
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-linux-arm64 $(MAIN_DIR)
	@$(call compress,$(DIST_DIR)/$(APP)-linux-arm64)
	@echo "   → $(DIST_DIR)/$(APP)-linux-arm64"

## build-darwin: compile for macOS amd64
build-darwin: web-dist
	@echo "🍎 darwin/amd64..."
	@mkdir -p $(DIST_DIR)
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-darwin-amd64 $(MAIN_DIR)
	@echo "   → $(DIST_DIR)/$(APP)-darwin-amd64"

## build-darwin-arm64: compile for macOS Apple Silicon
build-darwin-arm64: web-dist
	@echo "🍎 darwin/arm64..."
	@mkdir -p $(DIST_DIR)
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-darwin-arm64 $(MAIN_DIR)
	@echo "   → $(DIST_DIR)/$(APP)-darwin-arm64"

## build-windows: compile for Windows amd64
build-windows: web-dist
	@echo "🪟 windows/amd64..."
	@mkdir -p $(DIST_DIR)
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP)-windows-amd64.exe $(MAIN_DIR)
	@$(call compress,$(DIST_DIR)/$(APP)-windows-amd64.exe)
	@echo "   → $(DIST_DIR)/$(APP)-windows-amd64.exe"

## release: create release archives (binary + config.yaml + pages + frontend/build)
release: build-all
	@echo "📦 Creating release archives..."
	@rm -rf $(RELEASE_DIR) && mkdir -p $(RELEASE_DIR)
	@for p in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64; do \
		stage=$(RELEASE_DIR)/$(APP)-$$p; \
		mkdir -p $$stage; \
		cp $(DIST_DIR)/$(APP)-$$p $$stage/; \
		cp -r $(RELEASE_FILES) $$stage/; \
		(cd $(RELEASE_DIR) && tar -czf $(APP)-$$p.tar.gz $(APP)-$$p); \
		rm -rf $$stage; \
	done
	@stage=$(RELEASE_DIR)/$(APP)-windows-amd64; \
	mkdir -p $$stage; \
	cp $(DIST_DIR)/$(APP)-windows-amd64.exe $$stage/; \
	cp -r $(RELEASE_FILES) $$stage/; \
	(cd $(RELEASE_DIR) && tar -czf $(APP)-windows-amd64.tar.gz $(APP)-windows-amd64); \
	rm -rf $$stage
	@ls -lh $(RELEASE_DIR)

## clean: remove build artifacts
clean:
	@rm -rf $(DIST_DIR) $(RELEASE_DIR)
	@echo "🧹 Cleaned"

## clean-dist: remove old dist files
clean-dist:
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)
	@echo "🧹 Cleaned $(DIST_DIR)"

## help: show this help
help:
	@echo "$(APP) Build System"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
