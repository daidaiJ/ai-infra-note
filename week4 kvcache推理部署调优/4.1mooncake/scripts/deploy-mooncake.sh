#!/bin/bash
# ============================================================
# Mooncake PD分离部署脚本
# ============================================================
#
# 【用法】
# ./deploy-mooncake.sh <prefill_ip> <decode_ip> <model>
#
# 【示例】
# ./deploy-mooncake.sh 192.168.0.2 192.168.0.3 Qwen/Qwen2.5-7B-Instruct
#
# ============================================================

set -e

PREFILL_IP=${1:-"192.168.0.2"}
DECODE_IP=${2:-"192.168.0.3"}
MODEL=${3:-"Qwen/Qwen2.5-7B-Instruct"}
PROXY_PORT=${4:-"8000"}
PREFILL_PORT="8010"
DECODE_PORT="8020"
BOOTSTRAP_PORT="8998"

echo "============================================================"
echo "Mooncake PD分离部署"
echo "============================================================"
echo "Prefill节点: $PREFILL_IP:$PREFILL_PORT"
echo "Decode节点:  $DECODE_IP:$DECODE_PORT"
echo "Proxy端口:   $PROXY_PORT"
echo "模型:        $MODEL"
echo "============================================================"

# ============================================================
# Step1: 环境检查
# ============================================================
echo ""
echo "[Step1] 环境检查..."

# 检查RDMA设备
echo "检查RDMA设备..."
if ! ibv_devices 2>/dev/null; then
    echo "警告: RDMA设备检查失败，请确认IB/RoCE网卡已安装"
fi

# 检查numactl
echo "检查numactl..."
if ! command -v numactl &> /dev/null; then
    echo "警告: numactl未安装，NUMA绑定将无法执行"
fi

# ============================================================
# Step2: 获取网卡NUMA位置
# ============================================================
echo ""
echo "[Step2] NUMA拓扑检查..."

# 查找第一个RDMA网卡
NIC_NUMA=$(cat /sys/class/net/*/device/numa_node 2>/dev/null | head -1)
if [ -z "$NIC_NUMA" ] || [ "$NIC_NUMA" == "-1" ]; then
    NIC_NUMA=0
    echo "警告: 无法获取网卡NUMA，使用默认NUMA=0"
fi
echo "网卡NUMA位置: $NIC_NUMA"

# ============================================================
# Step3: 启动Prefill节点
# ============================================================
echo ""
echo "[Step3] 启动Prefill节点..."
echo "请在节点 $PREFILL_IP 执行以下命令："
echo ""
echo "export VLLM_MOONCAKE_BOOTSTRAP_PORT=$BOOTSTRAP_PORT"
echo "numactl --cpunodebind=$NIC_NUMA --membind=$NIC_NUMA \\"
echo "    vllm serve $MODEL \\"
echo "    --port $PREFILL_PORT \\"
echo "    --gpu-memory-utilization 0.85 \\"
echo "    --enable-prefix-caching \\"
echo "    --kv-transfer-config '{\"kv_connector\":\"MooncakeConnector\",\"kv_role\":\"kv_producer\"}'"
echo ""

# ============================================================
# Step4: 启动Decode节点
# ============================================================
echo "[Step4] 启动Decode节点..."
echo "请在节点 $DECODE_IP 执行以下命令："
echo ""
echo "numactl --cpunodebind=$NIC_NUMA --membind=$NIC_NUMA \\"
echo "    vllm serve $MODEL \\"
echo "    --port $DECODE_PORT \\"
echo "    --gpu-memory-utilization 0.85 \\"
echo "    --kv-transfer-config '{\"kv_connector\":\"MooncakeConnector\",\"kv_role\":\"kv_consumer\"}'"
echo ""

# ============================================================
# Step5: 启动Proxy代理
# ============================================================
echo "[Step5] 启动Proxy代理..."
echo "请在代理服务器执行以下命令："
echo ""
echo "python mooncake_connector_proxy.py \\"
echo "    --prefill http://$PREFILL_IP:$PREFILL_PORT \\"
echo "    --decode http://$DECODE_IP:$DECODE_PORT \\"
echo "    --port $PROXY_PORT"
echo ""

# ============================================================
# Step6: 验证部署
# ============================================================
echo "[Step6] 验证部署..."
echo "部署完成后，发送测试请求："
echo ""
echo "curl http://localhost:$PROXY_PORT/v1/completions \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"model\": \"$MODEL\", \"prompt\": \"Hello\"}'"
echo ""

echo "============================================================"
echo "部署说明完成"
echo "============================================================"
echo ""
echo "【重要提醒】"
echo "1. Prefill节点必须配置VLLM_MOONCAKE_BOOTSTRAP_PORT=$BOOTSTRAP_PORT"
echo "2. Proxy代理必须启动，否则请求无法路由"
echo "3. NUMA绑定是性能关键配置"
echo "4. 确认RDMA设备状态为PORT_ACTIVE"
echo ""