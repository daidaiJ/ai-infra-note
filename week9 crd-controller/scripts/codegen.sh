#!/bin/bash
# ============================================================
# 代码生成脚本：生成 DeepCopy 和客户端代码
# ============================================================
#
# 【功能】
# 1. 生成 DeepCopy 方法（zz_generated.deepcopy.go）
# 2. 生成 clientset/informers/listers 代码
#
# 【使用方式】
# ./scripts/codegen.sh
#
# 【前置条件】
# - Go 1.21+
# - controller-gen 工具
#
# ============================================================

set -euo pipefail

# ============================================================
# 配置变量
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Go 模块信息
MODULE="example.com/week9-crd-controller"
VERSION="v1alpha1"
GROUP="sample"

# ============================================================
# 颜色输出
# ============================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# ============================================================
# 步骤1: 检查前置条件
# ============================================================

info "检查前置条件..."

# 检查 Go
if ! command -v go &> /dev/null; then
    error "Go 未安装，请先安装 Go"
    exit 1
fi

info "Go 版本: $(go version)"

# 检查 controller-gen
if ! command -v controller-gen &> /dev/null; then
    warn "controller-gen 未安装，正在安装..."
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
fi

# ============================================================
# 步骤2: 生成 DeepCopy 代码
# ============================================================

info "生成 DeepCopy 代码..."

cd "$PROJECT_DIR"

controller-gen object:headerFile="" paths="./pkg/apis/..."

if [ $? -eq 0 ]; then
    info "DeepCopy 代码生成成功"
else
    error "DeepCopy 代码生成失败"
    exit 1
fi

# ============================================================
# 步骤3: 生成 CRD YAML 清单
# ============================================================

info "生成 CRD YAML 清单..."

controller-gen crd:crdVersions=v1 paths="./pkg/apis/..." output:crd:dir="$PROJECT_DIR/config/crd"

if [ $? -eq 0 ]; then
    info "CRD YAML 清单生成成功"
else
    error "CRD YAML 清单生成失败"
    exit 1
fi

# ============================================================
# 步骤4: 生成 RBAC 配置
# ============================================================

info "生成 RBAC 配置..."

controller-gen rbac:roleName=sample-controller-role paths="./pkg/controller/..." output:rbac:dir="$PROJECT_DIR/config/rbac"

if [ $? -eq 0 ]; then
    info "RBAC 配置生成成功"
else
    error "RBAC 配置生成失败"
    exit 1
fi

# ============================================================
# 步骤5: 格式化代码
# ============================================================

info "格式化代码..."

go fmt ./...

if [ $? -eq 0 ]; then
    info "代码格式化成功"
else
    error "代码格式化失败"
    exit 1
fi

# ============================================================
# 完成
# ============================================================

info "代码生成完成！"
info ""
info "生成的文件："
info "  - pkg/apis/sample/v1alpha1/zz_generated.deepcopy.go"
info "  - config/crd/sample.ai-infra_sampleresources.yaml"
info "  - config/rbac/role.yaml"
info ""
info "下一步："
info "  1. 查看生成的 CRD: cat config/crd/*.yaml"
info "  2. 部署到集群: kubectl apply -f config/crd/"
info "  3. 运行控制器: go run ./cmd/main.go"
