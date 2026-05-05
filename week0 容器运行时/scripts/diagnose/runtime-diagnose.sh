#!/bin/bash
# ============================================================
# runtime-diagnose.sh - 容器运行时诊断脚本
# ============================================================
#
# 【功能】
# - 检查运行时服务状态
# - 检查运行时版本和配置
# - 列出 Pod 和容器状态
# - 收集错误日志
#
# 【使用方式】
# chmod +x runtime-diagnose.sh
# ./runtime-diagnose.sh
#
# ============================================================

set -e

echo "=========================================="
echo "容器运行时诊断报告"
echo "生成时间: $(date)"
echo "主机名: $(hostname)"
echo "=========================================="

# ============================================================
# 1. 运行时服务状态
# ============================================================
echo ""
echo "=== 1. 运行时服务状态 ==="

# 检查 containerd
if systemctl is-active --quiet containerd; then
    echo "[containerd] 运行中 ✓"
    systemctl status containerd --no-pager -l | head -10
else
    echo "[containerd] 未运行或不存在"
fi

# 检查 CRI-O
if systemctl is-active --quiet crio; then
    echo "[CRI-O] 运行中 ✓"
    systemctl status crio --no-pager -l | head -10
else
    echo "[CRI-O] 未运行或不存在"
fi

# ============================================================
# 2. 运行时版本
# ============================================================
echo ""
echo "=== 2. 运行时版本 ==="

# 检查 crictl 版本
if command -v crictl &> /dev/null; then
    echo "[crictl 版本]"
    crictl version
else
    echo "[crictl] 未安装"
fi

# 检查 containerd 版本
if command -v containerd &> /dev/null; then
    echo "[containerd 版本]"
    containerd --version
fi

# 检查 CRI-O 版本
if command -v crio &> /dev/null; then
    echo "[CRI-O 版本]"
    crio --version
fi

# ============================================================
# 3. Socket 文件状态
# ============================================================
echo ""
echo "=== 3. Socket 文件状态 ==="

# containerd socket
if [ -S "/run/containerd/containerd.sock" ]; then
    echo "[containerd socket] 存在 ✓"
    ls -la /run/containerd/containerd.sock
else
    echo "[containerd socket] 不存在"
fi

# CRI-O socket
if [ -S "/run/crio/crio.sock" ]; then
    echo "[CRI-O socket] 存在 ✓"
    ls -la /run/crio/crio.sock
else
    echo "[CRI-O socket] 不存在"
fi

# ============================================================
# 4. 运行时信息
# ============================================================
echo ""
echo "=== 4. 运行时信息 ==="

if command -v crictl &> /dev/null; then
    # 配置 crictl
    if [ -S "/run/containerd/containerd.sock" ]; then
        export CRI_RUNTIME_ENDPOINT="/run/containerd/containerd.sock"
    elif [ -S "/run/crio/crio.sock" ]; then
        export CRI_RUNTIME_ENDPOINT="/run/crio/crio.sock"
    fi

    echo "[运行时信息]"
    crictl info 2>/dev/null || echo "无法获取运行时信息"
fi

# ============================================================
# 5. Pod 列表
# ============================================================
echo ""
echo "=== 5. Pod 列表 ==="

if command -v crictl &> /dev/null; then
    crictl pods 2>/dev/null || echo "无法获取 Pod 列表"
fi

# ============================================================
# 6. 容器列表
# ============================================================
echo ""
echo "=== 6. 容器列表 ==="

if command -v crictl &> /dev/null; then
    echo "[运行中的容器]"
    crictl ps 2>/dev/null || echo "无法获取容器列表"

    echo ""
    echo "[所有容器（含已停止）]"
    crictl ps -a 2>/dev/null || echo "无法获取容器列表"
fi

# ============================================================
# 7. 镜像列表
# ============================================================
echo ""
echo "=== 7. 镜像列表 ==="

if command -v crictl &> /dev/null; then
    crictl images 2>/dev/null || echo "无法获取镜像列表"
fi

# ============================================================
# 8. 镜像存储信息
# ============================================================
echo ""
echo "=== 8. 镜像存储信息 ==="

if command -v crictl &> /dev/null; then
    crictl imagefsinfo 2>/dev/null || echo "无法获取镜像存储信息"
fi

# ============================================================
# 9. 容器资源使用
# ============================================================
echo ""
echo "=== 9. 容器资源使用 ==="

if command -v crictl &> /dev/null; then
    crictl stats 2>/dev/null || echo "无法获取容器资源使用"
fi

# ============================================================
# 10. 最近错误日志
# ============================================================
echo ""
echo "=== 10. 最近错误日志 ==="

# containerd 错误日志
echo "[containerd 错误日志（最近1小时）]"
journalctl -u containerd -p err --since "1h ago" --no-pager 2>/dev/null | tail -20 || echo "无错误日志"

echo ""
echo "[CRI-O 错误日志（最近1小时）]"
journalctl -u crio -p err --since "1h ago" --no-pager 2>/dev/null | tail -20 || echo "无错误日志"

echo ""
echo "[kubelet 错误日志（最近1小时）]"
journalctl -u kubelet -p err --since "1h ago" --no-pager 2>/dev/null | tail -20 || echo "无错误日志"

# ============================================================
# 11. OOM 事件检查
# ============================================================
echo ""
echo "=== 11. OOM 事件检查 ==="

echo "[最近 OOM 事件]"
dmesg | grep -i "oom" | tail -10 2>/dev/null || echo "无 OOM 事件"

# ============================================================
# 12. 系统资源状态
# ============================================================
echo ""
echo "=== 12. 系统资源状态 ==="

echo "[磁盘空间]"
df -h /var/lib/containerd 2>/dev/null || df -h /var/lib/containers 2>/dev/null || df -h /

echo ""
echo "[内存使用]"
free -h

echo ""
echo "[CPU 负载]"
cat /proc/loadavg

# ============================================================
# 诊断完成
# ============================================================
echo ""
echo "=========================================="
echo "诊断报告完成"
echo "=========================================="

# ============================================================
# 常见问题提示
# ============================================================
echo ""
echo "【常见问题快速检查】"
echo ""
echo "1. Pod 无法启动？"
echo "   - 检查镜像是否存在: crictl images"
echo "   - 检查容器日志: crictl logs <container-id>"
echo "   - 检查运行时日志: journalctl -u containerd -f"
echo ""
echo "2. 容器频繁重启？"
echo "   - 检查退出码: kubectl describe pod <pod-name>"
echo "   - 检查 OOM: dmesg | grep oom"
echo "   - 检查资源限制: kubectl describe pod <pod-name>"
echo ""
echo "3. 镜像拉取失败？"
echo "   - 检查网络: curl -I https://registry-1.docker.io/v2/"
echo "   - 检查认证: kubectl get secrets"
echo "   - 手动拉取测试: crictl pull <image>"
echo ""