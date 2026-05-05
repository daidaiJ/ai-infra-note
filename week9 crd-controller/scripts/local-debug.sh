#!/bin/bash
# ============================================================
# 本地调试脚本：连接 Kind 集群并运行控制器
# ============================================================
#
# 【功能】
# 1. 检查 Kind 集群连接
# 2. 安装 CRD 到集群
# 3. 本地运行控制器（便于断点调试）
#
# 【使用方式】
# ./scripts/local-debug.sh
#
# 【前置条件】
# - Kind 集群已启动
# - kubectl 已配置
# - Go 1.21+
#
# ============================================================

set -euo pipefail

# ============================================================
# 配置变量
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Kind 集群名称
CLUSTER_NAME="${KIND_CLUSTER_NAME:-dev}"

# 日志级别（0=Info, 1=Debug, 2=Trace）
LOG_LEVEL="${LOG_LEVEL:-1}"

# ============================================================
# 颜色输出
# ============================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

debug() {
    if [ "$LOG_LEVEL" -ge 1 ]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# ============================================================
# 步骤1: 检查 Kind 集群
# ============================================================

info "检查 Kind 集群..."

if ! command -v kind &> /dev/null; then
    error "Kind 未安装，请先安装: go install sigs.k8s.io/kind@latest"
    exit 1
fi

# 检查集群是否存在
if ! kind get clusters | grep -q "$CLUSTER_NAME"; then
    warn "Kind 集群 '$CLUSTER_NAME' 不存在，正在创建..."
    kind create cluster --name "$CLUSTER_NAME"
fi

# 导出 KUBECONFIG
export KUBECONFIG=$(kind get kubeconfig --name "$CLUSTER_NAME")

info "已连接到 Kind 集群: $CLUSTER_NAME"

# 验证连接
debug "集群信息:"
kubectl cluster-info | head -2

# ============================================================
# 步骤2: 安装 CRD 到集群
# ============================================================

info "安装 CRD 到集群..."

cd "$PROJECT_DIR"

if [ -f "manifests/01-crd.yaml" ]; then
    kubectl apply -f manifests/01-crd.yaml
    
    # 等待 CRD 就绪
    info "等待 CRD 就绪..."
    kubectl wait --for=condition=Established crd/sampleresources.sample.ai-infra --timeout=30s
    
    info "CRD 已安装"
else
    error "CRD 清单文件不存在: manifests/01-crd.yaml"
    exit 1
fi

# ============================================================
# 步骤3: 安装 RBAC 配置
# ============================================================

info "安装 RBAC 配置..."

if [ -f "manifests/02-rbac.yaml" ]; then
    kubectl apply -f manifests/02-rbac.yaml
    info "RBAC 配置已安装"
else
    warn "RBAC 清单文件不存在: manifests/02-rbac.yaml"
fi

# ============================================================
# 步骤4: 验证 CRD
# ============================================================

info "验证 CRD..."

# 检查 CRD 是否注册
if kubectl get crd sampleresources.sample.ai-infra &> /dev/null; then
    info "CRD 已注册"
else
    error "CRD 未注册"
    exit 1
fi

# 检查 API 资源
debug "API 资源:"
kubectl api-resources | grep sampleresource || true

# ============================================================
# 步骤5: 运行控制器
# ============================================================

info "=========================================="
info "准备启动控制器..."
info "=========================================="
info ""
info "启动参数:"
info "  日志级别: --v=$LOG_LEVEL"
info "  指标地址: :8080"
info "  健康检查: :8081"
info ""
info "按 Ctrl+C 停止控制器"
info "=========================================="
info ""

# 构建并运行
cd "$PROJECT_DIR"

# 可选：使用 Delve 调试
if [ "${USE_DELVE:-false}" = "true" ]; then
    if ! command -v dlv &> /dev/null; then
        warn "Delve 未安装，正在安装..."
        go install github.com/go-delve/delve/cmd/dlv@latest
    fi
    
    info "使用 Delve 启动（监听 localhost:2345）..."
    dlv debug ./cmd/controller/main.go --headless --listen=:2345 --api-version=2 -- \
        --v=$LOG_LEVEL \
        --metrics-bind-address=:8080 \
        --health-probe-bind-address=:8081
else
    # 直接运行
    go run ./cmd/controller/main.go \
        --v=$LOG_LEVEL \
        --metrics-bind-address=:8080 \
        --health-probe-bind-address=:8081
fi
