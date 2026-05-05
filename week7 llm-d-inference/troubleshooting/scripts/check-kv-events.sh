#!/bin/bash
# ============================================================
# KV 事件验证脚本
# ============================================================
# 用法: ./check-kv-events.sh [namespace] [vllm-pod]

NAMESPACE=${1:-llm-d-inference}
VLLM_POD=${2:-vllm-0}

echo "=== vLLM Pod 状态 ==="
kubectl get pods -n ${NAMESPACE} -l app=vllm

echo ""
echo "=== KV Events 配置检查 ==="
kubectl exec -n ${NAMESPACE} ${VLLM_POD} -- env 2>/dev/null | grep -i "kv-events\|PYTHONHASHSEED" || echo "KV Events 未配置"

echo ""
echo "=== vLLM 日志中的 KV Events ==="
kubectl logs -n ${NAMESPACE} ${VLLM_POD} --tail=100 2>/dev/null | grep -i "kv_event\|block_store" || echo "无 KV Events 日志"

echo ""
echo "=== block-size 配置 ==="
kubectl logs -n ${NAMESPACE} ${VLLM_POD} --tail=500 2>/dev/null | grep -i "block-size\|block_size" || echo "未找到 block-size 配置"

echo ""
echo "=== 检查完成 ==="