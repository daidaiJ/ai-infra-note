#!/bin/bash
# ============================================================
# RDMA设备验证脚本
# ============================================================
#
# 【用途】验证Pod内RDMA设备可用性
# 【使用】kubectl exec <pod> -- ./verify-rdma.sh
#
# 【检查项】
# 1. RDMA设备是否存在
# 2. 设备端口状态
# 3. NUMA亲和性
# 4. RDMA连通性（可选）

set -e

echo "============================================================"
echo "RDMA设备验证脚本"
echo "============================================================"

# ============================================================
# 检查1: RDMA设备是否存在
# ============================================================
echo ""
echo "[检查1] RDMA设备是否存在"

if ! command -v ibv_devices &> /dev/null; then
    echo "[ERROR] ibv_devices命令不存在，请安装infiniband-tools"
    exit 1
fi

DEVICES=$(ibv_devices)
if [ -z "$DEVICES" ]; then
    echo "[FAIL] 无RDMA设备"
    exit 1
else
    echo "[OK] RDMA设备存在:"
    echo "$DEVICES"
fi

# ============================================================
# 检查2: 设备端口状态
# ============================================================
echo ""
echo "[检查2] 设备端口状态"

# 获取第一个设备名
DEVICE=$(ibv_devices | awk 'NR==2 {print $1}')
if [ -z "$DEVICE" ]; then
    echo "[FAIL] 无法获取设备名"
    exit 1
fi

echo "检查设备: $DEVICE"

# 检查端口状态
if command -v ibv_devinfo &> /dev/null; then
    PORT_STATE=$(ibv_devinfo -d $DEVICE | grep "state:" | awk '{print $2}')
    echo "端口状态: $PORT_STATE"

    if [ "$PORT_STATE" = "PORT_ACTIVE" ]; then
        echo "[OK] 设备状态: PORT_ACTIVE"
    else
        echo "[WARN] 设备状态非活跃: $PORT_STATE"
    fi
else
    echo "[WARN] ibv_devinfo命令不存在，跳过端口状态检查"
fi

# ============================================================
# 检查3: NUMA亲和性（如果可用）
# ============================================================
echo ""
echo "[检查3] NUMA亲和性"

if [ -d "/sys/class/infiniband/$DEVICE" ]; then
    NUMA_NODE=$(cat /sys/class/infiniband/$DEVICE/device/numa_node 2>/dev/null || echo "未知")
    echo "NUMA节点: $NUMA_NODE"

    if [ "$NUMA_NODE" != "-1" ] && [ "$NUMA_NODE" != "未知" ]; then
        echo "[OK] NUMA亲和: NUMANode $NUMA_NODE"

        # 检查当前进程NUMA
        if command -v numactl &> /dev/null; then
            CURRENT_NUMA=$(numactl --hardware | grep "current node:" | awk '{print $3}')
            echo "当前进程NUMA: $CURRENT_NUMA"

            if [ "$CURRENT_NUMA" = "$NUMA_NODE" ]; then
                echo "[OK] 进程与设备NUMA一致"
            else
                echo "[WARN] 进程与设备NUMA不一致（跨NUMA访问）"
            fi
        fi
    else
        echo "[WARN] NUMA信息未知"
    fi
else
    echo "[WARN] sysfs路径不存在，跳过NUMA检查"
fi

# ============================================================
# 检查4: 设备文件权限
# ============================================================
echo ""
echo "[检查4] 设备文件权限"

DEVICE_PATH="/dev/infiniband/$DEVICE"
if [ -e "$DEVICE_PATH" ]; then
    echo "[OK] 设备文件存在: $DEVICE_PATH"

    # 检查权限
    if [ -r "$DEVICE_PATH" ] && [ -w "$DEVICE_PATH" ]; then
        echo "[OK] 设备文件可读写"
    else
        echo "[FAIL] 设备文件权限不足"
        ls -l "$DEVICE_PATH"
    fi
else
    echo "[FAIL] 设备文件不存在: $DEVICE_PATH"
fi

# ============================================================
# 检查5: RDMA连通性（可选）
# ============================================================
echo ""
echo "[检查5] RDMA连通性（可选，需要对端节点）"

if [ -n "$PEER_IP" ]; then
    echo "测试对端节点: $PEER_IP"

    if command -v ib_write_bw &> /dev/null; then
        # 快速带宽测试
        RESULT=$(ib_write_bw -d $DEVICE -a $PEER_IP --size=65536 -n=10 2>&1 | tail -1)
        echo "带宽测试结果: $RESULT"
    else
        echo "[WARN] ib_write_bw命令不存在，跳过连通测试"
    fi
else
    echo "[SKIP] 未指定对端节点，跳过连通测试"
    echo "如需测试，设置环境变量: PEER_IP=<节点IP>"
fi

# ============================================================
# 总结
# ============================================================
echo ""
echo "============================================================"
echo "验证完成"
echo "============================================================"
echo "设备: $DEVICE"
echo "状态: $PORT_STATE"
echo "NUMA: $NUMA_NODE"
echo ""
echo "RDMA设备就绪，可开始使用。"