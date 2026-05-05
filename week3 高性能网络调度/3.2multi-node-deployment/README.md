# Multi-Node Deployment

> 多节点推理/训练服务的部署实践：网络配置、性能验证脚本、最佳实践示例。

---

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **完整学习文档** - 多节点部署方案与网络优化 |
| [scripts/verify-rdma.sh](scripts/verify-rdma.sh) | 🔧 RDMA设备验证脚本 |
| [scripts/benchmark-latency.sh](scripts/benchmark-latency.sh) | 🔧 延迟测试脚本 |
| [manifests/](manifests/) | 📋 部署配置清单 |

---

## 核心特性

```
┌─────────────────────────────────────────────────────────────────┐
│                 多节点部署核心能力                                │
├─────────────────────────────────────────────────────────────────┤
│  ✅ 网络配置     - NetworkAttachmentDefinition与Pod配置            │
│  ✅ 性能验证     - RDMA设备与连通性验证脚本                         │
│  ✅ 推理服务     - 多节点推理服务部署示例                           │
│  ✅ 训练任务     - 分布式训练NCCL配置示例                          │
│  ✅ 性能优化     - NUMA亲和与拓扑感知调度配置                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 项目结构

```
multi-node-deployment/
│
├── manifests/
│   ├── 01-network-attachment.yaml    # NAD网络配置
│   ├── 02-inference-service.yaml     # 推理服务部署
│   └── 03-training-job.yaml          # 训练任务部署
│
├── scripts/
│   ├── verify-rdma.sh               # RDMA验证脚本
│   └── benchmark-latency.sh         # 延迟测试脚本
│
├── docs/
│   └── README.md                    # 详细文档
│
└── README.md                        # 本文档
```

---

## 使用示例

### 多节点推理服务

```yaml
# ============================================================
# 示例: 多节点推理服务配置
# ============================================================
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inference-service
spec:
  replicas: 3              # 3节点部署
  template:
    spec:
      containers:
        - name: inference
          resources:
            limits:
              nvidia.com/gpu: 2
              rdma/ib: 1    # RDMA网卡
      # 拓扑感知调度
      topologySpreadConstraints:
        - topologyKey: kubernetes.io/hostname
```

### 分布式训练任务

```yaml
# ============================================================
# 示例: 分布式训练NCCL配置
# ============================================================
apiVersion: batch/v1
kind: Job
metadata:
  name: distributed-training
spec:
  parallelism: 4           # 4节点并行
  template:
    spec:
      containers:
        - name: pytorch
          env:
            - name: NCCL_DEBUG
              value: INFO
            - name: NCCL_IB_DISABLE
              value: "0"   # 启用RDMA
```

---

## 验证脚本

### RDMA 设备验证

```bash
# ============================================================
# 在Pod内验证RDMA设备
# ============================================================
./scripts/verify-rdma.sh

# 输出:
# [OK] RDMA设备存在: mlx5_0
# [OK] 设备状态: PORT_ACTIVE
# [OK] NUMA亲和: NUMANode 0
```

### 延迟测试

```bash
# ============================================================
# 测试节点间RDMA延迟
# ============================================================
./scripts/benchmark-latency.sh <peer-node-ip>

# 输出:
# 延迟测试结果: 2.5μs
# 帖宽测试结果: 95Gbps
```

---

详见 **[docs/README.md](docs/README.md)** 获取完整部署方案和网络优化配置。