# Network Topology Scheduler

> Kubernetes 调度器插件，实现 NUMA/机架/交换机维度的网络拓扑感知调度。

---

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **完整学习文档** - 拓扑感知调度算法与实现 |
| [pkg/plugin/topology.go](pkg/plugin/topology.go) | 调度插件核心实现 |
| [pkg/numa/numa.go](pkg/numa/numa.go) | NUMA亲和性算法 |
| [pkg/rack/rack.go](pkg/rack/rack.go) | 机架感知算法 |

---

## 核心特性

```
┌─────────────────────────────────────────────────────────────────┐
│                 网络拓扑感知调度核心能力                           │
├─────────────────────────────────────────────────────────────────┤
│  ✅ NUMA亲和      - 网卡与CPU同NUMA节点优先调度                   │
│  ✅ 机架感知      - 同机架节点优先（降低跨机架延迟）                │
│  ✅ 交换机感知    - 同交换机节点优先（降低网络跳数）                │
│  ✅ Filter过滤    - 过滤不符合拓扑要求的节点                       │
│  ✅ Score打分     - 根据拓扑距离打分，优先低延迟节点                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 项目结构

```
network-topology-scheduler/
│
├── pkg/
│   ├── plugin/topology.go    # 调度插件实现
│   ├── numa/numa.go         # NUMA亲和性
│   └── rack/rack.go         # 机架感知
│
├── config/
│   └── scheduler-config.yaml    # 调度器配置
│
├── docs/
│   └── README.md            # 详细文档
│
├── go.mod                   # Go模块定义
└── README.md                # 本文档
```

---

## 调度效果示例

### NUMA 亲和调度

```mermaid
graph TB
    subgraph "节点拓扑"
        N0["NUMA Node 0<br/>CPU 0-7<br/>网卡 mlx5_0"]
        N1["NUMA Node 1<br/>CPU 8-15<br/>网卡 mlx5_1"]
    end

    subgraph "Pod调度"
        P1["Pod请求<br/>rdma/ib + cpu"]
        P2["调度决策<br/>选择NUMA 0节点"]
    end

    P1 -->|"分析NUMA拓扑"| P2
    P2 -->|"调度到NUMA 0"| N0

    style N0 fill:#c8e6c9
    style P2 fill:#fff3e0
```

### 机架感知调度

```mermaid
graph TB
    subgraph "机架拓扑"
        R1["机架1<br/>Node1, Node2, Node3"]
        R2["机架2<br/>Node4, Node5"]
    end

    subgraph "多节点训练"
        T1["训练任务<br/>需要3个节点"]
        T2["调度决策<br/>优先同机架"]
    end

    T1 -->|"分析机架拓扑"| T2
    T2 -->|"选择机架1的3节点"| R1

    style R1 fill:#c8e6c9
    style T2 fill:#fff3e0
```

---

## 核心接口

```mermaid
graph TB
    subgraph "调度框架接口"
        F1["Filter<br/>过滤节点"]
        F2["Score<br/>打分排序"]
        F3["PreFilter<br/>预处理"]
    end

    subgraph "拓扑感知逻辑"
        T1["NUMA亲和检查"]
        T2["机架距离计算"]
        T3["交换机跳数计算"]
    end

    F1 --> T1
    F2 --> T2
    F2 --> T3

    style F1 fill:#e3f2fd
    style F2 fill:#c8e6c9
```

---

## 与Week1的知识复用

| Week1知识点 | 本项目应用 |
|-------------|-----------|
| Filter接口 | NUMA/拓扑条件过滤 |
| Score接口 | 拓扑距离打分算法 |
| 调度框架扩展 | 插件注册与配置 |

详见 **[docs/README.md](docs/README.md)** 获取完整实现细节。