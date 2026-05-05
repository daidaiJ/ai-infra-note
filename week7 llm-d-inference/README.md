# Week 7: llm-d 推理服务部署调优

> 系统学习 llm-d 推理服务栈的核心组件、调度扩展与自动缩放最佳实践。

---

## 学习系列总览

本项目涵盖 llm-d 推理服务栈的四大核心组件：

```mermaid
graph TB
    subgraph "控制平面"
        GW["Gateway<br/>Envoy/agentgateway"]
        EPP["Inference Scheduler<br/>(EPP) 推理调度器"]
        WVA["WVA<br/>自动缩放器"]
    end
    
    subgraph "数据平面"
        KV["KV-Cache Indexer<br/>缓存索引器"]
        MS["ModelService<br/>模型服务部署"]
    end
    
    subgraph "依赖组件"
        K8S["Kubernetes<br/>Gateway API/HPA"]
        PROM["Prometheus<br/>指标收集"]
        ZMQ["ZMQ<br/>KV事件传输"]
    end
    
    GW --> EPP --> KV
    EPP --> MS
    WVA --> MS
    KV --> ZMQ
    WVA --> PROM
    EPP --> PROM
    
    style EPP fill:#fff3e0
    style WVA fill:#e3f2fd
    style KV fill:#c8e6c9
    style MS fill:#fce4ec
```

---

## 快速导航

| 文件 | 说明 |
|------|------|
| [inference-scheduler/docs/README.md](inference-scheduler/docs/README.md) | 📖 **推理调度器详解** - EPP 插件架构、路由算法 |
| [autoscaler/docs/README.md](autoscaler/docs/README.md) | 📖 **自动缩放器详解** - WVA 饱和度驱动扩缩容 |
| [kv-cache/docs/README.md](kv-cache/docs/README.md) | 📖 **KV 缓存详解** - Indexer 双键设计、分层缓存 |
| [troubleshooting/docs/README.md](troubleshooting/docs/README.md) | 📖 **问题排查详解** - 分层排查模型 ⭐重点 |
| [architecture-summary/docs/README.md](architecture-summary/docs/README.md) | 📖 **架构总结详解** - 整体架构与关键原理 |

---

## 核心特性

```
┌─────────────────────────────────────────────────────────────┐
│                    llm-d 核心能力                            │
├─────────────────────────────────────────────────────────────┤
│  ✅ 智能路由   - EPP插件架构、前缀缓存感知、SLO感知调度       │
│  ✅ KV索引     - 双键设计、ZMQ事件订阅、分层缓存              │
│  ✅ 饱和度扩缩 - KV Cache+队列深度驱动、HPA/KEDA集成          │
│  ✅ P/D分离    - Prefill/Decode分离部署                      │
│  ✅ 多节点部署 - LeaderWorkerSet、Wide EP for MoE             │
└─────────────────────────────────────────────────────────────┘
```

---

## 项目结构

```
week5-llm-d-inference/
│
├── inference-scheduler/            # 推理调度器专题
│   ├── README.md                   # 概览：EPP 架构
│   ├── docs/                       # 详解：插件与路由算法
│   ├── pkg/                        # 插件示例代码
│   └── manifests/                  # 部署配置
│
├── autoscaler/                     # 自动缩放器专题
│   ├── README.md                   # 概览：WVA 架构
│   ├── docs/                       # 详解：饱和度扩缩容
│   └── manifests/                  # 部署配置
│
├── kv-cache/                       # KV 缓存专题
│   ├── README.md                   # 概览：Indexer 架构
│   ├── docs/                       # 详解：双键设计
│   └── manifests/                  # 配置示例
│
├── troubleshooting/                # 问题排查 ⭐重点
│   ├── README.md                   # 概览：排查框架
│   ├── docs/                       # 详解：分层排查
│   └── scripts/                    # 诊断脚本
│
├── architecture-summary/           # 架构总结
│   ├── README.md                   # 概览：整体架构
│   ├── docs/                       # 详解：关键原理
│
└── docs/                           # 通用文档
    ├── gateway-api-gie.md          # Gateway API Inference Extension
    └── best-practices.md           # 最佳实践
```

---

## 核心组件依赖关系

```mermaid
graph TB
    subgraph "Kubernetes 原生机制"
        K1["Gateway API<br/>InferencePool/HTTPRoute"]
        K2["HPA/KEDA<br/>水平自动扩缩"]
        K3["Controller Runtime<br/>自定义控制器"]
        K4["LeaderWorkerSet<br/>多节点部署"]
        K5["DRA<br/>动态资源分配"]
    end
    
    subgraph "Envoy 机制"
        E1["ext-proc<br/>外部处理器回调"]
        E2["FULL_DUPLEX_STREAMED<br/>双向流模式"]
    end
    
    subgraph "llm-d 核心组件"
        L1["llm-d-inference-scheduler"]
        L2["llm-d-kv-cache"]
        L3["llm-d-workload-variant-autoscaler"]
        L4["llm-d-modelservice"]
    end
    
    subgraph "外部依赖"
        X1["ZMQ (go-zeromq/zmq4)<br/>KV事件传输"]
        X2["Prometheus<br/>指标收集"]
        X3["vLLM/SGLang<br/>推理引擎"]
        X4["Redis/Valkey<br/>可选索引存储"]
    end
    
    K1 --> L1
    E1 --> L1
    L1 --> L2
    X1 --> L2
    K2 --> L3
    X2 --> L3
    L3 --> L1
    K4 --> L4
    K5 --> L4
    X3 --> L4
    
    style L1 fill:#fff3e0
    style L2 fill:#c8e6c9
    style L3 fill:#e3f2fd
    style L4 fill:#fce4ec
```

---

## 学习路线

### 阶段一：理解核心组件

```mermaid
graph LR
    S1["Inference Scheduler<br/>EPP插件架构"] --> S2["KV-Cache Indexer<br/>双键设计"]
    S2 --> S3["WVA Autoscaler<br/>饱和度扩缩"]
    
    style S1 fill:#fff3e0
    style S2 fill:#c8e6c9
    style S3 fill:#e3f2fd
```

### 阶段二：掌握部署与调优

```mermaid
graph LR
    S1["部署模式选择"] --> S2["配置调优"]
    S2 --> S3["问题排查"]
    
    style S3 fill:#ffcdd2
```

### 阶段三：架构总结

```mermaid
graph LR
    S1["整体架构理解"] --> S2["关键实现原理"]
    S2 --> S3["与其他方案对比"]
    
    style S1 fill:#c8e6c9
```

---

## 核心收获

| 能力维度 | 收获 |
|----------|------|
| **调度器扩展** | EPP 插件架构（Filters/Scorers/Pickers）、自定义插件开发 |
| **KV 缓存索引** | 双键设计、ZMQ 事件订阅、分层缓存机制 |
| **自动缩放器** | WVA 饱和度驱动机制、与 EPP 阈值对齐、HPA/KEDA 集成 |
| **问题排查** | 分层排查方法（Gateway→EPP→模型服务→网络） |

---

## 源码项目

| 项目 | 链接 | 研读重点 |
|------|------|----------|
| llm-d-inference-scheduler | `llm-d/llm-d-inference-scheduler` | EPP 插件架构、路由流程 |
| llm-d-kv-cache | `llm-d/llm-d-kv-cache` | Indexer 实现、双键设计 |
| llm-d-workload-variant-autoscaler | `llm-d/llm-d-workload-variant-autoscaler` | 饱和度分析器、HPA 集成 |
| llm-d-modelservice | `llm-d/llm-d-modelservice` | Helm Chart 结构 |

---

> 开始学习：推荐从 [inference-scheduler/docs/README.md](inference-scheduler/docs/README.md) 开始。