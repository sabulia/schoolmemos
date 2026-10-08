#!/usr/bin/env bash
# ============================================================
# 校园Memo 一键安装脚本（Linux / macOS）
#
# 用法（任选其一）：
#   curl -fsSL https://raw.githubusercontent.com/<你的用户名>/campus-memos/main/scripts/install-campus.sh | bash
#   bash scripts/install-campus.sh
#
# 策略：有 Docker 用 Docker（最省事）；没有 Docker 就下载单二进制；
# 都没有则提示先安装其一。装完打印本机访问地址，非技术同学也能跑起来。
# ============================================================
set -euo pipefail

APP_NAME="campus-memos"
DATA_DIR="${CAMPUS_DATA:-$HOME/.campus-memos}"
PORT="${CAMPUS_PORT:-5230}"
# TODO(fork 后修改为你的仓库地址)
REPO="${CAMPUS_REPO:-yourname/campus-memos}"
VERSION="${CAMPUS_VERSION:-latest}"

echo "==> 校园Memo 一键安装"
echo "    数据目录: $DATA_DIR"
echo "    访问端口: $PORT"
mkdir -p "$DATA_DIR"

if command -v docker >/dev/null 2>&1; then
  echo "==> 检测到 Docker，使用容器方式安装"
  docker pull "ghcr.io/${REPO}:${VERSION}" || docker pull "ghcr.io/${REPO}:latest"
  docker rm -f "$APP_NAME" >/dev/null 2>&1 || true
  docker run -d --name "$APP_NAME" \
    -p "${PORT}:5230" \
    -v "${DATA_DIR}:/var/opt/memos" \
    --restart unless-stopped \
    "ghcr.io/${REPO}:latest"
  echo "==> 完成！浏览器打开: http://localhost:${PORT}"
  exit 0
fi

echo "==> 未检测到 Docker，尝试下载单二进制（无需安装数据库）"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "不支持的架构: $ARCH"; exit 1 ;;
esac

URL="https://github.com/${REPO}/releases/${VERSION}/download/campus-memos_${OS}_${ARCH}.tar.gz"
echo "    下载: $URL"
if curl -fsSL "$URL" -o /tmp/campus-memos.tar.gz; then
  tar -xzf /tmp/campus-memos.tar.gz -C /tmp
  install -m 0755 /tmp/campus-memos "$HOME/.local/bin/campus-memos" 2>/dev/null || {
    mkdir -p "$HOME/bin"; install -m 0755 /tmp/campus-memos "$HOME/bin/campus-memos"
  }
  BIN="$(command -v campus-memos || echo "$HOME/bin/campus-memos")"
  echo "==> 启动服务（后台运行）"
  nohup "$BIN" --data "$DATA_DIR" --port "$PORT" > "$DATA_DIR/run.log" 2>&1 &
  echo "==> 完成！浏览器打开: http://localhost:${PORT}"
  echo "    停止: pkill campus-memos；数据在 $DATA_DIR"
else
  echo "下载失败。请先安装 Docker（https://docs.docker.com/get-docker/）后重跑本脚本，"
  echo "或到 https://github.com/${REPO}/releases 手动下载对应平台的二进制。"
  exit 1
fi
