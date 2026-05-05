# Architecture Summary (架构总结)

> llm-d 整体架构、四大核心组件协作关系、关键实现原理。

---

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **核心架构详解** - 关键实现原理 |

---

## 整体架构

```mermaid
graph TB
    subgraph "请求入口"
        CLIENT["客户端"]
        GW["Gateway<br/>Envoy/agentgateway"]
    end
    
    subgraph "控制平面"
        EPP["Inference Scheduler<br/>(EPP)"]
        WVA["WVA<br/>Autoscaler"]
    end
    
    subgraph "数据平面"
        KV["KV-Cache Indexer"]
        VLLM["vLLM Pods"]
    end
    
    subgraph "支撑组件"
        PROM["Prometheus"]
        ZMQ["ZMQ"]
        HPA["HPA/KEDA"]
    end
    
    CLIENT --> GW --> EPP --> VLLM --> CLIENT
    KV --> EPP
    VLLM -->|"KV Events"| ZMQ --> KV
    VLLM -->|"Metrics"| PROM --> WVA --> HPA --> VLLM
    
    style EPP fill:#fff3e0
    style WVA fill:#e3f2fd
    style KV fill:#c8e6c9
    style VLLM fill:#fce4ec
```

---

## 四大核心组件

| 组件 | 核心职责 | 关键依赖 |
|------|----------|----------|
| **Inference Scheduler** | 推理请求智能路由 | Gateway API, ext-proc, KV Indexer |
| **KV-Cache Indexer** | 全局 KV 缓存状态索引 | ZMQ, vLLM KV Events |
| **WVA Autoscaler** | 饱和度驱动自动扩缩 | Prometheus, HPA/KEDA |
| **ModelService** | 模型服务声明式部署 | Helm, vLLM, LWS |

---

## 关键实现原理

### 1. ext-proc 回调机制

```mermaid
sequenceDiagram
    participant Envoy as Envoy Gateway
    participant EPP as EPP (ext-proc)
    
    Envoy->>EPP: gRPC ProcessingRequest (FULL_DUPLEX_STREAMED)
    Note over EPP: 执行调度决策<br/>Filters → Scorers → Picker
    EPP-->>Envoy: ProcessingResponse (目标 Pod 地址)
    Envoy->>Envoy: 转发请求到目标 Pod
```

### 2. 双键索引设计

```mermaid
graph LR
    subgraph "Write Path"
        E1["BlockStored<br/>engine_key + tokens"]
        R1["计算 request_key"]
        MAP["建立映射<br/>engine_key → request_key"]
    end
    
    subgraph "Read Path"
        T["Prompt Tokens"]
        R2["计算 request_key"]
        Q["查询 Index"]
    end
    
    E1 --> R1 --> MAP
    T --> R2 --> Q
    
    style MAP fill:#c8e6c9
    style Q fill:#fff3e0
```

### 3. 饱和度驱动扩缩

```mermaid
graph TB
    subgraph "指标收集"
        M1["KV Cache 利用率"]
        M2["队列深度"]
    end
    
    subgraph "决策逻辑"
        S["计算 spare capacity"]
        T["avg_spare < trigger?"]
        D["输出 wva_desired_replicas"]
    end
    
    subgraph "执行扩缩"
        H["HPA/KEDA"]
        P["调整 Pod 数量"]
    end
    
    M1 --> S --> T --> D --> H --> P
    M2 --> S
    
    style T fill:#fff3e0
    style D fill:#c8e6c9
```

---

## 项目结构

```
architecture-summary/
├── README.md                   # 本文档
├── docs/
│   ├── README.md               # 核心架构详解
│   ├── component-overview.md   # 组件协作关系
│   ├── key-implementations.md  # 关键实现原理
│   └── comparison.md           # 与其他方案对比
```

---

详见 **[docs/README.md](docs/README.md)** 获取核心架构详解。