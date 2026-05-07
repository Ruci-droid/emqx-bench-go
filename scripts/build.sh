#!/usr/bin/env bash
# 跨平台编译与打包脚本
# 用法:
#   ./scripts/build.sh              # 编译所有平台并打包
#   ./scripts/build.sh linux        # 仅 Linux 平台 (amd64, arm64, arm32)
#   ./scripts/build.sh windows      # 仅 Windows 平台 (amd64)
#   ./scripts/build.sh all          # 所有平台
#   ./scripts/build.sh package      # 编译并打包为 tar.gz / zip

set -euo pipefail

BINARY="go-mqtt-bench"
BUILD_DIR="build"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
LDFLAGS="-s -w -X main.version=${VERSION}"

# 颜色输出
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m' # 无颜色

log()  { echo -e "${GREEN}[build]${NC} $*"; }
info() { echo -e "${CYAN}  ->${NC} $*"; }

# 编译单个平台
# $1: GOOS
# $2: GOARCH
# $3: 输出文件名
# $4: 额外 GO 环境变量 (可选, 如 GOARM=7)
build_target() {
    local os="$1"
    local arch="$2"
    local output="$3"
    local extra_env="${4:-}"

    log "编译 ${os}/${arch} ..."
    mkdir -p "${BUILD_DIR}"

    if [ -n "${extra_env}" ]; then
        env GOOS="${os}" GOARCH="${arch}" "${extra_env}" \
            go build -trimpath -ldflags="${LDFLAGS}" -o "${BUILD_DIR}/${output}" .
    else
        env GOOS="${os}" GOARCH="${arch}" \
            go build -trimpath -ldflags="${LDFLAGS}" -o "${BUILD_DIR}/${output}" .
    fi

    info "${BUILD_DIR}/${output}"
}

# 打包：Linux/macOS 用 tar.gz，Windows 用 zip
package_target() {
    local file="$1"
    local full_path="${BUILD_DIR}/${file}"

    if [[ ! -f "${full_path}" ]]; then
        return
    fi

    if [[ "${file}" == *.exe ]]; then
        log "打包 ${file} -> ${file}.zip"
        zip -j "${full_path}.zip" "${full_path}" > /dev/null
        rm -f "${full_path}"
        info "${full_path}.zip"
    else
        log "打包 ${file} -> ${file}.tar.gz"
        tar -czf "${full_path}.tar.gz" -C "${BUILD_DIR}" "${file}"
        rm -f "${full_path}"
        info "${full_path}.tar.gz"
    fi
}

build_linux() {
    build_target linux amd64 "${BINARY}-linux-amd64"
    build_target linux arm64 "${BINARY}-linux-arm64"
    build_target linux arm   "${BINARY}-linux-arm32" GOARM=7
}

build_windows() {
    build_target windows amd64 "${BINARY}-windows-amd64.exe"
}

build_darwin() {
    build_target darwin amd64 "${BINARY}-darwin-amd64"
    build_target darwin arm64 "${BINARY}-darwin-arm64"
}

build_all() {
    build_linux
    build_windows
    build_darwin
}

do_package() {
    log "打包所有产物..."
    for f in "${BUILD_DIR}/${BINARY}"-*; do
        [ -f "$f" ] || continue
        package_target "$(basename "$f")"
    done
    log "打包完成"
}

print_summary() {
    echo ""
    echo "========================================"
    echo "  go-mqtt-bench 构建结果"
    echo "  版本: ${VERSION}"
    echo "========================================"
    ls -lh "${BUILD_DIR}"/ 2>/dev/null || echo "  (无产物)"
    echo ""
}

case "${1:-all}" in
    linux)
        build_linux
        ;;
    windows)
        build_windows
        ;;
    darwin|macos)
        build_darwin
        ;;
    all)
        build_all
        ;;
    package)
        build_all
        do_package
        ;;
    *)
        echo "用法: $0 {linux|windows|darwin|all|package}"
        exit 1
        ;;
esac

print_summary
