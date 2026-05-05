#!/bin/bash
# ============================================================
# EPP 健康检查脚本
# ============================================================
# 用法: ./check-epp-health.sh [namespace] [epp-name]

NAMESPACE=${1:-llm-d-inference}
EPP_NAME=${2:-gaie-epp}

echo "=== EPP Pod 状态 ==="
kubectl get pods -n ${NAMESPACE} -l app=epp

echo ""
echo "=== EPP 日志 (错误和警告) ==="
kubectl logs -n ${NAMESPACE} deployment/${EPP_NAME} --tail=50 | grep -i "error\|warn" || echo "无错误日志"

echo ""
echo "=== ZMQ 连接状态 ==="
kubectl exec -n ${NAMESPACE} deployment/${EPP_NAME} -- netstat -an 2>/dev/null | grep 5556 || echo "ZMQ 端口未监听"

echo ""
echo "=== 插件配置关键参数 ==="
kubectl get cm -n ${NAMESPACE} -o yaml 2>/dev/null | grep -A3 "blockSize\|hashSeed\|modelName" || echo "ConfigMap 未找到"

echo ""
echo "=== Tokenizer Sidecar 状态 ==="
kubectl get pods -n ${NAMESPACE} deployment/${EPP_NAME} -o jsonpath='{.spec.containers[*].name}' 2>/dev/null

echo ""
echo "=== 检查完成 ==="