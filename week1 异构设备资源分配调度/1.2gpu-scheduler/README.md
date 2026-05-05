# GPU-Aware Kubernetes Scheduler

一个自定义 Kubernetes 调度器，实现 **GPU 利用率均衡** 和 **设备 UUID 精确绑定** 功能。

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **完整学习文档** - 包含调度器架构、插件接口详解、Mermaid 流程图 |
| [pkg/gpuinfo/gpuinfo.go](pkg/gpuinfo/gpuinfo.go) | GPU 信息获取 Mock 接口 |
| [pkg/plugins/gpuutilization/gpuutilization.go](pkg/plugins/gpuutilization/gpuutilization.go) | GPU 利用率均衡插件 |
| [pkg/plugins/deviceuuid/deviceuuid.go](pkg/plugins/deviceuuid/deviceuuid.go) | 设备 UUID 匹配插件 |
| [cmd/scheduler/main.go](cmd/scheduler/main.go) | 调度器入口 |
| [config/scheduler-config.yaml](config/scheduler-config.yaml) | 调度器配置 |
| [manifests/deployment.yaml](manifests/deployment.yaml) | K8s 部署清单 |

## 核心特性

```
┌─────────────────────────────────────────────────────────┐
│                    GPU Scheduler                         │
├─────────────────────────────────────────────────────────┤
│  ✅ GPUUtilizationBalancer  - 优先调度到低负载节点        │
│  ✅ DeviceUUIDMatcher       - 绑定到指定 GPU 设备        │
└─────────────────────────────────────────────────────────┘
```

## 项目结构

```
gpu-scheduler/
├── cmd/scheduler/main.go           # 入口
├── pkg/
│   ├── gpuinfo/                    # GPU 信息 Mock
│   └── plugins/
│       ├── gpuutilization/         # 利用率均衡插件
│       └── deviceuuid/             # UUID 匹配插件
├── config/                         # 配置文件
├── manifests/                      # K8s 清单
└── docs/README.md                  # 详细文档
```

## 使用示例

```yaml
# 指定 GPU 设备
apiVersion: v1
kind: Pod
metadata:
  name: my-gpu-pod
  annotations:
    scheduler.example.com/target-gpu-uuid: "GPU-abc123-0001"
spec:
  schedulerName: gpu-scheduler      # 使用 GPU 调度器
  containers:
    - name: app
      image: nvidia/cuda:12.0
      resources:
        limits:
          nvidia.com/gpu: 1
```

详见 **[docs/README.md](docs/README.md)** 获取完整架构说明和使用指南。