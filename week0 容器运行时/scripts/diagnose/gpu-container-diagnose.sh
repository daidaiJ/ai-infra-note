#!/bin/bash
# ============================================================
# gpu-container-diagnose.sh - GPU 容器诊断脚本
# ============================================================
#
# 【功能】
# - 检查宿主机 GPU 状态
# - 检查 NVIDIA 驱动和 Toolkit 配置
# - 检查 CDI 文件状态
# - 诊断容器 GPU 配置
#
# 【使用方式】
# chmod +x gpu-container-diagnose.sh
# ./gpu-container-diagnose.sh [pod-name] [namespace]
#
# ============================================================

set -e

POD_NAME=$1
NAMESPACE=$2

if [ -z "$NAMESPACE" ]; then
    NAMESPACE="default"
fi

echo "=========================================="
echo "GPU 容器诊断报告"
echo "生成时间: $(date)"
echo "主机名: $(hostname)"
echo "=========================================="

# ============================================================
# 1. 宿主机 GPU 状态
# ============================================================
echo ""
echo "=== 1. 宿主机 GPU 状态 ==="

if command -v nvidia-smi &> /dev/null; then
    echo "[GPU 设备列表]"
    nvidia-smi -L 2>/dev/null || echo "无法获取 GPU 列表"

    echo ""
    echo "[GPU 详细状态]"
    nvidia-smi 2>/dev/null || echo "无法获取 GPU 状态"
else
    echo "[警告] nvidia-smi 未安装，请检查 NVIDIA 驱动"
fi

echo ""
echo "[GPU 设备文件]"
ls -la /dev/nvidia* 2>/dev/null || echo "/dev/nvidia* 设备文件不存在"

# ============================================================
# 2. NVIDIA 驱动版本
# ============================================================
echo ""
echo "=== 2. NVIDIA 驱动版本 ==="

if [ -f "/proc/driver/nvidia/version" ]; then
    cat /proc/driver/nvidia/version
else
    echo "[警告] /proc/driver/nvidia/version 不存在，驱动可能未加载"
fi

# ============================================================
# 3. NVIDIA Container Toolkit
# ============================================================
echo ""
echo "=== 3. NVIDIA Container Toolkit ==="

# 检查 nvidia-container-cli
if command -v nvidia-container-cli &> /dev/null; then
    echo "[nvidia-container-cli 信息]"
    nvidia-container-cli info 2>/dev/null || echo "无法获取 Toolkit 信息"

    echo ""
    echo "[nvidia-container-cli 版本]"
    nvidia-container-cli --version 2>/dev/null || echo "无法获取版本"
else
    echo "[警告] nvidia-container-cli 未安装"
fi

# 检查 nvidia-ctk
if command -v nvidia-ctk &> /dev/null; then
    echo ""
    echo "[nvidia-ctk 版本]"
    nvidia-ctk --version 2>/dev/null || echo "无法获取版本"

    echo ""
    echo "[nvidia-ctk 验证]"
    nvidia-ctk validate 2>/dev/null || echo "验证失败或未配置"
else
    echo "[警告] nvidia-ctk 未安装"
fi

# ============================================================
# 4. Toolkit 配置文件
# ============================================================
echo ""
echo "=== 4. Toolkit 配置文件 ==="

if [ -f "/etc/nvidia-container-runtime/config.toml" ]; then
    echo "[配置文件内容]"
    cat /etc/nvidia-container-runtime/config.toml | head -50
else
    echo "[警告] 配置文件不存在: /etc/nvidia-container-runtime/config.toml"
fi

# ============================================================
# 5. CDI 文件状态
# ============================================================
echo ""
echo "=== 5. CDI 文件状态 ==="

echo "[CDI 目录内容]"
ls -la /etc/cdi/ 2>/dev/null || echo "CDI 目录不存在"

if [ -f "/etc/cdi/nvidia.yaml" ]; then
    echo ""
    echo "[CDI 文件内容（前 50 行）]"
    head -50 /etc/cdi/nvidia.yaml

    echo ""
    echo "[CDI 设备数量]"
    grep -c "^  - name:" /etc/cdi/nvidia.yaml 2>/dev/null || echo "无法统计"
else
    echo "[警告] CDI 文件不存在: /etc/cdi/nvidia.yaml"
    echo "请执行: nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml"
fi

# ============================================================
# 6. 运行时 GPU 配置
# ============================================================
echo ""
echo "=== 6. 运行时 GPU 配置 ==="

if command -v crictl &> /dev/null; then
    # 检查运行时配置中的 GPU
    echo "[containerd GPU 运行时配置]"
    if [ -f "/etc/containerd/config.toml" ]; then
        grep -A 5 "nvidia" /etc/containerd/config.toml 2>/dev/null || echo "未找到 nvidia 运行时配置"
    fi

    echo ""
    echo "[CRI-O GPU 运行时配置]"
    if [ -f "/etc/crio/crio.conf" ]; then
        grep -A 5 "nvidia" /etc/crio/crio.conf 2>/dev/null || echo "未找到 nvidia 运行时配置"
    fi

    echo ""
    echo "[运行时信息]"
    crictl info 2>/dev/null | grep -i "nvidia" || echo "运行时信息中未包含 GPU 配置"
fi

# ============================================================
# 7. NVIDIA Device Plugin 状态
# ============================================================
echo ""
echo "=== 7. NVIDIA Device Plugin 状态 ==="

if command -v kubectl &> /dev/null; then
    echo "[Device Plugin Pod]"
    kubectl get pods -n kube-system -l name=nvidia-device-plugin-ds 2>/dev/null || \
        kubectl get pods -n kube-system | grep nvidia 2>/dev/null || \
        echo "未找到 NVIDIA Device Plugin Pod"

    echo ""
    echo "[Device Plugin 日志（最近 20 行）]"
    PLUGIN_POD=$(kubectl get pods -n kube-system -l name=nvidia-device-plugin-ds -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$PLUGIN_POD" ]; then
        kubectl logs -n kube-system $PLUGIN_POD --tail 20 2>/dev/null || echo "无法获取日志"
    fi

    echo ""
    echo "[节点 GPU 资源]"
    kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU:.status.capacity.nvidia\\.com/gpu 2>/dev/null || \
        echo "无法获取节点 GPU 资源"
fi

# ============================================================
# 8. 容器 GPU 配置诊断（如果指定了 Pod）
# ============================================================
if [ -n "$POD_NAME" ]; then
    echo ""
    echo "=== 8. Pod '$POD_NAME' GPU 配置诊断 ==="

    if command -v kubectl &> /dev/null; then
        echo "[Pod 基本信息]"
        kubectl get pod $POD_NAME -n $NAMESPACE -o wide 2>/dev/null || echo "Pod 不存在"

        echo ""
        echo "[Pod GPU 资源请求]"
        kubectl describe pod $POD_NAME -n $NAMESPACE 2>/dev/null | grep -A 5 "nvidia.com/gpu" || \
            echo "Pod 未请求 GPU 资源"

        echo ""
        echo "[Pod 环境变量]"
        kubectl describe pod $POD_NAME -n $NAMESPACE 2>/dev/null | grep -A 20 "Environment:" | grep -i "nvidia" || \
            echo "Pod 无 NVIDIA 相关环境变量"

        echo ""
        echo "[Pod 事件]"
        kubectl describe pod $POD_NAME -n $NAMESPACE 2>/dev/null | grep -A 10 "Events:" || \
            echo "无事件信息"
    fi

    # 检查运行时容器配置
    if command -v crictl &> /dev/null; then
        echo ""
        echo "[运行时容器状态]"
        CONTAINER_ID=$(crictl ps | grep $POD_NAME | awk '{print $1}' 2>/dev/null)
        if [ -n "$CONTAINER_ID" ]; then
            echo "容器 ID: $CONTAINER_ID"

            echo ""
            echo "[设备挂载]"
            crictl inspect $CONTAINER_ID 2>/dev/null | grep -A 10 "devices" | grep nvidia || \
                echo "容器无 NVIDIA 设备挂载"

            echo ""
            echo "[挂载点]"
            crictl inspect $CONTAINER_ID 2>/dev/null | grep -A 3 "nvidia" || \
                echo "容器无 NVIDIA 相关挂载点"

            echo ""
            echo "[环境变量]"
            crictl inspect $CONTAINER_ID 2>/dev/null | grep -i "NVIDIA_VISIBLE_DEVICES" || \
                echo "容器无 NVIDIA_VISIBLE_DEVICES 环境变量"
        else
            echo "[警告] Pod 对应的容器未找到"
        fi
    fi
fi

# ============================================================
# 9. CUDA 兼容性检查
# ============================================================
echo ""
echo "=== 9. CUDA 兼容性检查 ==="

if command -v nvidia-smi &> /dev/null; then
    DRIVER_VERSION=$(nvidia-smi | grep "Driver Version" | awk '{print $6}' 2>/dev/null)
    CUDA_VERSION=$(nvidia-smi | grep "CUDA Version" | awk '{print $9}' 2>/dev/null)

    echo "[驱动版本] $DRIVER_VERSION"
    echo "[CUDA 版本] $CUDA_VERSION"

    echo ""
    echo "[CUDA 与驱动兼容性参考]"
    echo "CUDA 12.x 需要 Driver >= 525.60.13"
    echo "CUDA 11.x 需要 Driver >= 450.80.02"
fi

# ============================================================
# 10. 常见 GPU 问题提示
# ============================================================
echo ""
echo "=========================================="
echo "诊断报告完成"
echo "=========================================="

echo ""
echo "【常见 GPU 容器问题检查清单】"
echo ""
echo "1. GPU 设备不可见？"
echo "   - 检查驱动: nvidia-smi"
echo "   - 检查设备文件: ls -la /dev/nvidia*"
echo "   - 检查 Toolkit: nvidia-container-cli info"
echo "   - 检查运行时配置: crictl info | grep nvidia"
echo ""
echo "2. NVIDIA_VISIBLE_DEVICES 不生效？"
echo "   - 检查运行时模式: /etc/nvidia-container-runtime/config.toml"
echo "   - CDI 模式下由 CDI 文件管理，忽略此变量"
echo "   - 检查 Device Plugin 是否正常运行"
echo ""
echo "3. CUDA 库找不到？"
echo "   - 检查容器挂载: crictl inspect <container-id>"
echo "   - 检查 NVIDIA_DRIVER_CAPABILITIES"
echo "   - 使用官方 CUDA 镜像"
echo ""
echo "4. 驱动版本不匹配？"
echo "   - 检查宿主机驱动: nvidia-smi"
echo "   - 检查容器 CUDA 版本: kubectl exec <pod> -- nvcc --version"
echo "   - 参考 CUDA 兼容性表"
echo ""
echo "5. CDI 配置问题？"
echo "   - 生成 CDI 文件: nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml"
echo "   - 配置运行时: nvidia-ctk runtime configure --runtime=containerd"
echo "   - 重启运行时: systemctl restart containerd"
echo ""