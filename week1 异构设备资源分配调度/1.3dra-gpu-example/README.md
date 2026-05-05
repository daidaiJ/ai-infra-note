# Kubernetes DRA (Dynamic Resource Allocation) 示例

> 展示 Kubernetes 1.26+ 新的 DRA 机制的典型资源分配声明示例。

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **完整学习文档** - DRA三大组件协作流程、与Device Plugin对比 |
| [manifests/01-resourceclass.yaml](manifests/01-resourceclass.yaml) | ResourceClass 资源类定义 |
| [manifests/02-resourceclaim.yaml](manifests/02-resourceclaim.yaml) | ResourceClaim 资源声明 |
| [manifests/03-pod.yaml](manifests/03-pod.yaml) | Pod/Deployment 使用 DRA |
| [manifests/04-driver.yaml](manifests/04-driver.yaml) | DRA Driver 部署概念 |

## DRA 三大核心组件

```
┌─────────────────────────────────────────────────────────────────┐
│                     DRA 核心组件                                 │
├─────────────────────────────────────────────────────────────────┤
│  ResourceClass      → 资源类定义（配置、驱动、参数）              │
│  ResourceClaim      → 资源声明（请求、约束、生命周期）            │
│  ResourceSlice      → 节点设备信息（属性、拓扑、状态）            │
├─────────────────────────────────────────────────────────────────┤
│  DRA Driver         → 驱动实现（发现、分配、准备、清理）          │
└─────────────────────────────────────────────────────────────────┘
```

## 项目结构

```
dra-gpu-example/
├── manifests/
│   ├── 01-resourceclass.yaml    # ResourceClass + 参数 ConfigMap
│   ├── 02-resourceclaim.yaml    # ResourceClaim 多种示例
│   ├── 03-pod.yaml              # Pod/Deployment/StatefulSet
│   └── 04-driver.yaml           # Driver DaemonSet + ResourceSlice
└── docs/README.md               # 详细学习文档
```

## DRA vs Device Plugin

| 能力 | Device Plugin | DRA |
|------|---------------|-----|
| 约束匹配 | ❌ 仅数量 | ✅ CEL表达式 |
| 生命周期 | ❌ Pod绑定 | ✅ 独立管理 |
| 跨节点资源 | ❌ 不支持 | ✅ 支持 |
| 资源共享 | ❌ 独占 | ✅ shareable |
| 拓扑感知 | ⚠️ 有限 | ✅ 完整 |

详见 **[docs/README.md](docs/README.md)** 获取完整流程和最佳实践。