#!/bin/bash
# ============================================================
# RDMA延迟测试脚本
# ============================================================
#
# 【用途】测试节点间RDMA延迟和带宽
# 【使用】
#   服务端: ./benchmark-latency.sh -s
#   客户端: ./benchmark-latency.sh -c <server-ip>
#
# 【测试项】
# 1. 延迟测试（ib_write_lat）
# 2. 带宽测试（ib_write_bw）
# 3. 性能基线对比

set -e

# ============================================================
# 参数解析
# ============================================================

MODE=""
SERVER_IP=""
DEVICE="mlx5_0"
TEST_SIZE="65536"

while getopts "s:c:d:h" opt; do
    case $opt in
        s)
            MODE="server"
            ;;
        c)
            MODE="client"
            SERVER_IP="$OPTARG"
            ;;
        d)
            DEVICE="$OPTARG"
            ;;
        h)
            echo "使用方法:"
            echo "  服务端: $0 -s"
            echo "  客户端: $0 -c <server-ip>"
            echo "  可选参数:"
            echo "    -d <device>  指定RDMA设备（默认: mlx5_0）"
            exit 0
            ;;
        \?)
            echo "无效参数: -$OPTARG"
            exit 1
            ;;
    esac
done

# ============================================================
# 工具检查
# ============================================================

check_tools() {
    echo "============================================================"
    echo "检查测试工具"
    echo "============================================================"

    if ! command -v ib_write_lat &> /dev/null; then
        echo "[ERROR] ib_write_lat不存在，请安装perftest工具包"
        exit 1
    fi

    if ! command -v ib_write_bw &> /dev/null; then
        echo "[ERROR] ib_write_bw不存在，请安装perftest工具包"
        exit 1
    fi

    echo "[OK] 测试工具可用"
}

# ============================================================
# 设备检查
# ============================================================

check_device() {
    echo ""
    echo "============================================================"
    echo "检查RDMA设备"
    echo "============================================================"

    if ! ibv_devices | grep -q "$DEVICE"; then
        echo "[ERROR] 设备 $DEVICE 不存在"
        ibv_devices
        exit 1
    fi

    echo "[OK] 设备 $DEVICE 存在"

    # 显示设备信息
    echo ""
    echo "设备详情:"
    ibv_devinfo -d $DEVICE | grep -E "state|mtu|max_msg_sz"
}

# ============================================================
# 延迟测试
# ============================================================

test_latency() {
    echo ""
    echo "============================================================"
    echo "延迟测试（ib_write_lat）"
    echo "============================================================"

    if [ "$MODE" = "server" ]; then
        echo "启动服务端..."
        ib_write_lat -d $DEVICE -a -F
    elif [ "$MODE" = "client" ]; then
        echo "连接服务端: $SERVER_IP"
        RESULT=$(ib_write_lat -d $DEVICE -a $SERVER_IP -F 2>&1)

        echo ""
        echo "测试结果:"
        echo "$RESULT"

        # ============================================================
        # 解析延迟值
        # ============================================================
        LATENCY=$(echo "$RESULT" | grep "64" | awk '{print $4}')
        if [ -n "$LATENCY" ]; then
            echo ""
            echo "============================================================"
            echo "延迟分析"
            echo "============================================================"
            echo "典型延迟: $LATENCY μs"

            # 性能基线对比
            if [ "$LATENCY" -lt 3 ]; then
                echo "[EXCEL] 性能优秀（< 3μs）"
            elif [ "$LATENCY" -lt 10 ]; then
                echo "[GOOD] 性能良好（< 10μs）"
            elif [ "$LATENCY" -lt 50 ]; then
                echo "[WARN] 性能一般（< 50μs），可能有MTU或NUMA问题"
            else
                echo "[BAD] 性能较差（> 50μs），需要排查"
            fi
        fi
    fi
}

# ============================================================
# 帖宽测试
# ============================================================

test_bandwidth() {
    echo ""
    echo "============================================================"
    echo "带宽测试（ib_write_bw）"
    echo "============================================================"

    if [ "$MODE" = "server" ]; then
        echo "启动服务端..."
        ib_write_bw -d $DEVICE -a -F --size=$TEST_SIZE
    elif [ "$MODE" = "client" ]; then
        echo "连接服务端: $SERVER_IP"
        RESULT=$(ib_write_bw -d $DEVICE -a $SERVER_IP -F --size=$TEST_SIZE 2>&1)

        echo ""
        echo "测试结果:"
        echo "$RESULT"

        # ============================================================
        # 解析带宽值
        # ============================================================
        BW_MB=$(echo "$RESULT" | grep "$TEST_SIZE" | awk '{print $3}')
        if [ -n "$BW_MB" ]; then
            BW_GB=$(echo "$BW_MB * 8 / 1000" | bc -l | awk '{printf "%.1f", $0}')

            echo ""
            echo "============================================================"
            echo "带宽分析"
            echo "============================================================"
            echo "带宽: $BW_MB MB/s ≈ $BW_GB Gbps"

            # 性能基线对比（假设100G网卡）
            if [ "$(echo "$BW_GB > 90" | bc)" -eq 1 ]; then
                echo "[EXCEL] 性能优秀（> 90Gbps）"
            elif [ "$(echo "$BW_GB > 80" | bc)" -eq 1 ]; then
                echo "[GOOD] 性能良好（> 80Gbps）"
            elif [ "$(echo "$BW_GB > 50" | bc)" -eq 1 ]; then
                echo "[WARN] 性能一般（> 50Gbps），可能有PCIe或NUMA瓶颈"
            else
                echo "[BAD] 性能较差（< 50Gbps），需要排查"
            fi
        fi
    fi
}

# ============================================================
# 主流程
# ============================================================

if [ -z "$MODE" ]; then
    echo "请指定运行模式:"
    echo "  服务端: $0 -s"
    echo "  客户端: $0 -c <server-ip>"
    exit 1
fi

check_tools
check_device

if [ "$MODE" = "server" ]; then
    echo ""
    echo "============================================================"
    echo "运行模式: 服务端"
    echo "============================================================"
    echo "请在客户端运行: $0 -c <本机IP>"
    echo ""

    # 同时启动延迟和带宽服务端（不同端口）
    test_latency
elif [ "$MODE" = "client" ]; then
    echo ""
    echo "============================================================"
    echo "运行模式: 客户端"
    echo "============================================================"
    echo "目标服务端: $SERVER_IP"
    echo "测试设备: $DEVICE"
    echo ""

    test_latency
    # test_bandwidth  # 可选：运行带宽测试
fi

# ============================================================
# 性能基线参考
# ============================================================

echo ""
echo "============================================================"
echo "性能基线参考"
echo "============================================================"
echo "IB 100G网卡基线:"
echo "  延迟: ~1μs"
echo "  帖宽: ~95Gbps"
echo ""
echo "RoCE 100G网卡基线:"
echo "  延迟: ~2μs"
echo "  帖宽: ~90Gbps"
echo ""
echo "如性能不达基线，请参考性能问题排查指南进行诊断。"