.PHONY: build test lint release release-linux release-windows release-all package clean

BINARY     := go-mqtt-bench
BUILD_DIR  := build
GO         := go
GOFLAGS    := -trimpath -ldflags="-s -w"
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -s -w -X main.version=$(VERSION)

# 默认编译当前平台
build:
	$(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .

test:
	$(GO) test -v -race ./...

lint:
	@which golangci-lint > /dev/null && golangci-lint run ./... || echo "未安装 golangci-lint，跳过 lint"

# -----------------------------------------------------------
# 按平台编译
# -----------------------------------------------------------

# Linux
release-linux-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .

release-linux-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .

release-linux-arm32:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm32 .

# Linux 全架构
release-linux: release-linux-amd64 release-linux-arm64 release-linux-arm32

# Windows
release-windows-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe .

# macOS (额外赠送)
release-darwin-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .

release-darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .

release-darwin: release-darwin-amd64 release-darwin-arm64

# 全平台
release-all: release-linux release-windows-amd64 release-darwin
	@echo "全部平台编译完成，产物在 $(BUILD_DIR)/ 目录"
	@ls -lh $(BUILD_DIR)/

# -----------------------------------------------------------
# 打包（tar.gz / zip）
# -----------------------------------------------------------
package: release-all
	@echo "打包中..."
	@for f in $(BUILD_DIR)/$(BINARY)-linux-*; do \
		tar -czf "$$f.tar.gz" -C $(BUILD_DIR) "$$(basename $$f)"; \
		echo "  $$f.tar.gz"; \
	done
	@for f in $(BUILD_DIR)/$(BINARY)-darwin-*; do \
		tar -czf "$$f.tar.gz" -C $(BUILD_DIR) "$$(basename $$f)"; \
		echo "  $$f.tar.gz"; \
	done
	@for f in $(BUILD_DIR)/$(BINARY)-windows-*.exe; do \
		zip -j "$$f.zip" "$$f" > /dev/null; \
		echo "  $$f.zip"; \
	done
	@echo "打包完成"

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)
