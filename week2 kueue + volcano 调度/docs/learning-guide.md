# AI任务调度学习指导文档

> 系统梳理 Kueue + Volcano 知识体系，明确学习重点、开源项目指引和分层学习路径。

---

## 目录

- [1. 知识体系总览](#1-知识体系总览)
- [2. 知识点分类矩阵](#2-知识点分类矩阵)
- [3. 开源项目学习指引](#3-开源项目学习指引)
- [4. 分层学习建议](#4-分层学习建议)
- [5. Kueue核心知识点](#5-kueue核心知识点)
- [6. Volcano核心知识点](#6-volcano核心知识点)
- [7. 结合方案核心知识点](#7-结合方案核心知识点)
- [8. 业务场景映射](#8-业务场景映射)
- [9. 学习路线图](#9-学习路线图)
- [附录](#附录)

---

## 1. 知识体系总览

### 1.1 技术栈定位

```mermaid
graph TB
    subgraph "基础设施层"
        I1["Kubernetes<br/>容器编排平台"]
        I2["GPU设备<br/>NVIDIA/AMD等"]
    end

    subgraph "资源分配层 (Week1)"
        R1["Device Plugin<br/>设备发现"]
        R2["自定义调度器<br/>GPU感知"]
        R3["DRA<br/>声明式资源"]
    end

    subgraph "任务调度层 (Week2)"
        T1["Kueue<br/>队列管理"]
        T2["Volcano<br/>批处理调度"]
    end

    subgraph "应用层"
        A1["训练任务<br/>PyTorch/TensorFlow"]
        A2["推理服务<br/>模型部署"]
        A3["开发调试<br/>交互式任务"]
    end

    I1 --> R1 --> T1 --> A1
    I2 --> R2 --> T2 --> A2
    I1 --> R3 --> T1 --> A3

    style R1 fill:#fff3e0
    style R2 fill:#e3f2fd
    style R3 fill:#c8e6c9
    style T1 fill:#2196f3
    style T2 fill:#ff9800
```

### 1.2 Week1与Week2的关系

```mermaid
graph LR
    subgraph "Week1: 解决资源如何分配"
        W1Q["问题: GPU设备如何被Pod使用？"]
        W1A["答案: Device Plugin分配 → DRA声明"]
    end

    subgraph "Week2: 解决任务如何调度"
        W2Q["问题: 多任务如何公平使用资源？"]
        W2A["答案: Kueue队列管理 → Volcano批调度"]
    end

    W1Q --> W1A --> W2Q --> W2A

    style W1Q fill:#ffcdd2
    style W1A fill:#c8e6c9
    style W2Q fill:#fff3e0
    style W2A fill:#2196f3
```

| 维度 | Week1 资源分配 | Week2 任务调度 |
|------|---------------|---------------|
| **核心问题** | GPU设备如何分配给Pod | 多任务如何公平使用GPU |
| **解决层面** | 设备层（节点本地） | 任务层（集群全局） |
| **关键技术** | Device Plugin、DRA、调度框架 | Kueue、Volcano |
| **业务价值** | 使GPU可被容器使用 | 使GPU资源高效利用 |

---

## 2. 知识点分类矩阵

### 2.1 四级分类体系

```mermaid
quadrantChart
    title AI任务调度知识点学习分类
    x-axis "理解原理" --> "动手实现"
    y-axis "必须深入" --> "学会使用"

    quadrant-1 "源码研读区"
    quadrant-2 "原理深究区"
    quadrant-3 "实践应用区"
    quadrant-4 "配置使用区"

    "队列分配算法": [0.90, 0.85]
    "Gang调度实现": [0.85, 0.80]
    "抢占机制逻辑": [0.80, 0.75]
    "公平分享算法": [0.75, 0.70]

    "队列模型设计": [0.30, 0.80]
    "配额模型原理": [0.35, 0.75]
    "调度器扩展点": [0.40, 0.85]

    "优先级策略": [0.25, 0.40]
    "任务依赖配置": [0.20, 0.35]
    "ResourceFlavor": [0.15, 0.30]

    "历史演进背景": [0.50, 0.20]
    "多集群架构": [0.60, 0.25]
```

### 2.2 详细分类表

#### 🔴 必须深入实现（源码研读区）

| 知识点 | 所属项目 | 研读重点 | 学习价值 |
|--------|----------|----------|----------|
| **队列分配算法** | Kueue | `scheduler.go` 分配逻辑 | 理解如何选择最优队列分配 |
| **抢占机制实现** | Kueue | `preemption.go` 抢占逻辑 | 理解高优先级任务如何获取资源 |
| **Gang调度实现** | Volcano | `gang_scheduler.go` | 理解All-or-Nothing调度语义 |
| **公平分享算法** | Volcano | `share_scheduler.go` | 理解DRF/SSJ公平分配算法 |
| **调度器插件扩展** | Volcano | `plugins/` 目录 | 理解如何扩展调度能力 |

**源码研读路径：**

```mermaid
graph TB
    subgraph "Kueue源码研读"
        K1["scheduler/scheduler.go<br/>分配入口"]
        K2["preemption/preemption.go<br/>抢占逻辑"]
        K3["queue/cluster_queue.go<br/>队列状态管理"]
    end

    subgraph "Volcano源码研读"
        V1["pkg/scheduler/plugins/gang<br/>Gang调度插件"]
        V2["pkg/scheduler/plugins/share<br/>公平分享插件"]
        V3["pkg/scheduler/framework<br/>调度框架"]
    end

    K1 --> K2 --> K3
    V1 --> V2 --> V3

    style K1 fill:#2196f3
    style K2 fill:#2196f3
    style K3 fill:#2196f3
    style V1 fill:#ff9800
    style V2 fill:#ff9800
    style V3 fill:#ff9800
```

#### 🟠 需要理解原理（原理深究区）

| 知识点 | 原理说明 | 理解重点 |
|--------|----------|----------|
| **队列模型设计** | ClusterQueue + LocalQueue两级模型 | 理解为什么要分两级 |
| **配额模型原理** | nominalQuota + borrowingLimit | 理解配额与借用的关系 |
| **Cohort机制** | 队列组共享资源池 | 理解跨队列资源共享 |
| **PodGroup语义** | 一组Pod必须同时调度 | 理解分布式训练调度需求 |
| **优先级模型** | PriorityClass + preempting | 理解抢占决策逻辑 |

**原理理解要点：**

```mermaid
graph TB
    subgraph "Kueue队列模型原理"
        Q1["两级队列设计<br/>ClusterQueue(集群) + LocalQueue(命名空间)"]
        Q2["为什么两级？<br/>集群级配额控制 + 命名空间级提交入口"]
        Q3["配额模型<br/>nominalQuota(固定) + borrowing(借用)"]
    end

    subgraph "Volcano Gang调度原理"
        G1["PodGroup概念<br/>一组相关Pod的调度单元"]
        G2["minAvailable语义<br/>必须同时可用的Pod数"]
        G3["为什么需要？<br/>分布式训练通信依赖"]
    end

    Q1 --> Q2 --> Q3
    G1 --> G2 --> G3

    style Q1 fill:#2196f3
    style Q2 fill:#2196f3
    style Q3 fill:#2196f3
    style G1 fill:#ff9800
    style G2 fill:#ff9800
    style G3 fill:#ff9800
```

#### 🟢 学会配置使用（配置使用区）

| 配置项 | 所属项目 | 使用场景 | 配置重点 |
|--------|----------|----------|----------|
| **ClusterQueue配置** | Kueue | 团队配额管理 | nominalQuota、flavor配置 |
| **LocalQueue配置** | Kueue | 命名空间提交入口 | 关联ClusterQueue |
| **Workload提交** | Kueue | 任务提交 | resourceRequests、优先级 |
| **Queue配置** | Volcano | 队列能力分配 | weight、capability |
| **Job配置** | Volcano | 批处理任务 | minAvailable、tasks |
| **PodGroup配置** | Volcano | Gang调度 | minMember、queue |

**配置示例速查：**

```yaml
# ============================================================
# Kueue核心配置要素
# ============================================================
# ClusterQueue: 团队配额
spec:
  resourceGroups:
    - flavors:
        - resources:
            - name: "nvidia.com/gpu"
              nominalQuota: 10        # 固定配额
              borrowingLimit: 5       # 可借用上限

# ============================================================
# Volcano核心配置要素
# ============================================================
# Job: 批处理任务
spec:
  minAvailable: 4                     # Gang调度：必须4个Pod
  queue: "training-queue"             # 指定队列
  tasks:
    - replicas: 4                     # Pod数量
```

#### ⚪ 认知了解（背景了解区）

| 内容 | 了解价值 | 学习方式 |
|------|----------|----------|
| **历史演进背景** | 理解技术发展脉络 | 阅读官方博客、论文 |
| **多集群调度架构** | 了解跨集群调度方案 | 阅读Kueue多集群文档 |
| **其他调度系统对比** | 了解YARN/Slurm等 | 对比分析文章 |
| **社区发展趋势** | 了解未来发展方向 | 关注CNCF项目动态 |

---

## 3. 开源项目学习指引

### 3.1 项目分类

```mermaid
graph TB
    subgraph "需要研读源码"
        S1["Kueue<br/>队列管理核心"]
        S2["Volcano<br/>批处理调度核心"]
    end

    subgraph "需要学会使用"
        U1["volcano-kueue-integration<br/>集成方案"]
        U2["Job Operators<br/>MPIJob/TFJob"]
    end

    subgraph "认知了解"
        K1["Kueue多集群<br/>跨集群调度"]
        K2["Volcano历史<br/>演进背景"]
    end

    S1 --> S2 --> U1 --> U2 --> K1 --> K2

    style S1 fill:#ffcdd2
    style S2 fill:#ffcdd2
    style U1 fill:#c8e6c9
    style U2 fill:#c8e6c9
    style K1 fill:#9e9e9e
    style K2 fill:#9e9e9e
```

### 3.2 详细指引表

#### 需要研读源码的项目

| 项目 | GitHub链接 | 研读重点 | 研读哪些部分 |
|------|------------|----------|-------------|
| **Kueue** | https://github.com/kubernetes-sigs/kueue | 队列分配、抢占机制 | `pkg/scheduler/`, `pkg/preemption/` |
| **Volcano** | https://github.com/volcano-sh/volcano | Gang调度、公平分享 | `pkg/scheduler/plugins/` |

**Kueue源码研读清单：**

| 文件路径 | 研读内容 | 解决的问题 |
|----------|----------|------------|
| `pkg/scheduler/scheduler.go` | 分配算法入口 | 理解Workload如何被分配到ClusterQueue |
| `pkg/preemption/preemption.go` | 抢占逻辑实现 | 理解高优先级任务如何抢占资源 |
| `pkg/queue/cluster_queue.go` | 队列状态管理 | 理解配额计算和状态维护 |
| `pkg/workload/workload.go` | Workload生命周期 | 理解任务状态变迁 |
| `apis/kueue/v1beta1/` | CRD定义 | 理解API设计思路 |

**Volcano源码研读清单：**

| 文件路径 | 研读内容 | 解决的问题 |
|----------|----------|------------|
| `pkg/scheduler/plugins/gang/gang.go` | Gang调度插件 | 理解All-or-Nothing调度实现 |
| `pkg/scheduler/plugins/share/share.go` | 公平分享插件 | 理解DRF/SSJ算法实现 |
| `pkg/scheduler/plugins/priority/` | 优先级插件 | 理解优先级调度逻辑 |
| `pkg/scheduler/framework/framework.go` | 调度框架 | 理解插件扩展点设计 |
| `apis/batch/v1alpha1/` | Job CRD定义 | 理解批处理Job API设计 |

#### 需要学会使用的项目

| 项目 | GitHub链接 | 使用场景 | 学习重点 |
|------|------------|----------|----------|
| **volcano-kueue-integration** | https://github.com/volcano-sh/volcano | Volcano集成Kueue | 配置方式、兼容性处理 |
| **MPIJob Operator** | https://github.com/kubeflow/mpi-operator | MPI分布式训练 | Job提交、配置参数 |
| **TFJob Operator** | https://github.com/kubeflow/tf-training-operator | TensorFlow训练 | Job提交、Worker/PS配置 |

#### 认知了解的内容

| 内容 | 了解价值 | 参考资源 |
|------|----------|----------|
| **Kueue多集群调度** | 了解跨集群资源调度方案 | Kueue官方多集群文档 |
| **Volcano演进历史** | 理解批处理调度发展脉络 | Volcano设计文档 |
| **调度系统对比** | 了解不同调度系统特点 | YARN vs Kubernetes对比文章 |

---

## 4. 分层学习建议

### 4.1 四层学习模型

```mermaid
graph TB
    subgraph "原理层（必须理解）"
        P1["队列管理本质<br/>任务排队 → 资源分配"]
        P2["批调度本质<br/>一组任务协同调度"]
        P3["公平分享本质<br/>按需求比例分配"]
    end

    subgraph "接口层（需要掌握）"
        I1["Kueue CRD API<br/>ClusterQueue/LocalQueue"]
        I2["Volcano CRD API<br/>Queue/Job/PodGroup"]
        I3["调度器插件接口<br/>扩展点设计"]
    end

    subgraph "实现层（源码研读）"
        R1["分配算法实现<br/>如何选择最优队列"]
        R2["抢占机制实现<br/>如何决定抢占目标"]
        R3["Gang调度实现<br/>如何保证同时调度"]
    end

    subgraph "应用层（学会使用）"
        A1["配置部署<br/>队列配额设置"]
        A2["任务提交<br/>Workload/Job创建"]
        A3["问题排查<br/>调度失败分析"]
    end

    P1 --> I1 --> R1 --> A1
    P2 --> I2 --> R2 --> A2
    P3 --> I3 --> R3 --> A3

    style P1 fill:#4caf50
    style P2 fill:#4caf50
    style P3 fill:#4caf50
    style I1 fill:#2196f3
    style I2 fill:#2196f3
    style I3 fill:#2196f3
    style R1 fill:#ff9800
    style R2 fill:#ff9800
    style R3 fill:#ff9800
    style A1 fill:#c8e6c9
    style A2 fill:#c8e6c9
    style A3 fill:#c8e6c9
```

### 4.2 学习优先级

```mermaid
graph TB
    subgraph "高优先级（必须掌握）"
        H1["队列分配逻辑<br/>Kueue核心"]
        H2["Gang调度语义<br/>Volcano核心"]
        H3["抢占机制<br/>高优先级任务处理"]
    end

    subgraph "中优先级（需要理解）"
        M1["配额模型<br/>nominalQuota机制"]
        M2["公平分享算法<br/>DRF/SSJ原理"]
        M3["PodGroup设计<br/>批处理语义"]
    end

    subgraph "低优先级（学会使用）"
        L1["队列配置<br/>ClusterQueue YAML"]
        L2["Job配置<br/>批处理任务参数"]
        L3["优先级配置<br/>PriorityClass设置"]
    end

    style H1 fill:#ffcdd2
    style H2 fill:#ffcdd2
    style H3 fill:#ffcdd2
    style M1 fill:#fff3e0
    style M2 fill:#fff3e0
    style M3 fill:#fff3e0
    style L1 fill:#c8e6c9
    style L2 fill:#c8e6c9
    style L3 fill:#c8e6c9
```

### 4.3 各层级学习方式

| 学习层级 | 推荐方式 | 时间占比 | 目标 |
|----------|----------|----------|------|
| **原理层** | 阅读设计文档、思考本质问题 | 20% | 理解设计动机和核心概念 |
| **接口层** | 阅读API定义、编写示例配置 | 25% | 掌握CRD结构和配置方法 |
| **实现层** | 研读源码、理解关键算法 | 35% | 理解核心实现细节 |
| **应用层** | 实际部署、调试问题 | 20% | 解决实际问题 |

---

## 5. Kueue核心知识点

### 5.1 组件架构

```mermaid
graph TB
    subgraph "Kueue核心组件"
        CQ["ClusterQueue<br/>集群级队列（配额）"]
        LQ["LocalQueue<br/>命名空间级队列"]
        WL["Workload<br/>任务抽象"]
        RF["ResourceFlavor<br/>资源类型标签"]
        CO["Cohort<br/>队列组（共享池）"]
    end

    subgraph "关系"
        WL --> LQ --> CQ
        CQ --> RF
        CQ --> CO
    end

    style CQ fill:#2196f3
    style LQ fill:#2196f3
    style WL fill:#ff9800
    style RF fill:#c8e6c9
    style CO fill:#9e9e9e
```

### 5.2 知识点详解

| 知识点 | 学习分类 | 学习重点 | 解决的问题 |
|--------|----------|----------|------------|
| **ClusterQueue** | 🟢 配置使用 | 配额配置、flavor选择 | 团队级别资源配额 |
| **LocalQueue** | 🟢 配置使用 | 关联ClusterQueue | 命名空间提交入口 |
| **Workload** | 🟠 理解原理 | 状态变迁、优先级 | 任务抽象和调度状态 |
| **队列分配算法** | 🔴 源码研读 | 分配决策逻辑 | 如何选择最优队列 |
| **抢占机制** | 🔴 源码研读 | 抢占目标选择 | 高优先级任务获取资源 |
| **Cohort共享** | 🟠 理解原理 | 跨队列借用机制 | 团队间资源共享 |
| **ResourceFlavor** | 🟢 配置使用 | 资源类型标签 | 区分不同GPU类型 |

### 5.3 核心流程

```mermaid
sequenceDiagram
    participant User as 用户
    participant LQ as LocalQueue
    participant CQ as ClusterQueue
    participant Scheduler as Kueue Scheduler
    participant K8s as Kubernetes

    Note over User,K8s: === 任务提交流程 ===

    User->>LQ: 提交Workload
    LQ->>CQ: 查找关联ClusterQueue
    CQ->>Scheduler: 检查配额可用性

    Note over Scheduler: 配额充足？
    Scheduler->>Scheduler: 是：直接分配
    Scheduler->>Scheduler: 否：排队等待

    Scheduler->>K8s: 创建Pod
    K8s-->>User: Pod运行

    Note over User,K8s: === 抢占流程 ===

    User->>LQ: 提交高优先级Workload
    Scheduler->>Scheduler: 配额不足，触发抢占
    Scheduler->>K8s: 删除低优先级Pod
    Scheduler->>K8s: 创建高优先级Pod
```

---

## 6. Volcano核心知识点

### 6.1 组件架构

```mermaid
graph TB
    subgraph "Volcano核心组件"
        Q["Queue<br/>队列（权重/能力）"]
        J["Job<br/>批处理任务"]
        PG["PodGroup<br/>调度组"]
        MPI["MPIJob<br/>MPI训练"]
        TF["TFJob<br/>TensorFlow训练"]
    end

    subgraph "调度插件"
        GANG["Gang插件<br/>All-or-Nothing"]
        SHARE["Share插件<br/>公平分享"]
        PRI["Priority插件<br/>优先级"]
    end

    J --> PG --> GANG
    PG --> SHARE
    PG --> PRI

    MPI --> J
    TF --> J

    style Q fill:#ff9800
    style J fill:#ff9800
    style PG fill:#ff9800
    style GANG fill:#ffcdd2
    style SHARE fill:#ffcdd2
    style PRI fill:#ffcdd2
```

### 6.2 知识点详解

| 知识点 | 学习分类 | 学习重点 | 解决的问题 |
|--------|----------|----------|------------|
| **Queue** | 🟢 配置使用 | 权重、能力配置 | 队列级别资源分配 |
| **Job** | 🟢 配置使用 | tasks、minAvailable | 批处理任务定义 |
| **PodGroup** | 🔴 源码研读 | minMember、调度状态 | Gang调度核心语义 |
| **Gang调度** | 🔴 源码研读 | 算法实现 | All-or-Nothing调度保证 |
| **公平分享** | 🔴 源码研读 | DRF/SSJ算法 | 按需求比例公平分配 |
| **调度框架** | 🔴 源码研读 | 插件扩展点 | 如何扩展调度能力 |
| **MPIJob/TFJob** | 🟢 学会使用 | 配置和提交 | 分布式训练任务 |

### 6.3 Gang调度核心流程

```mermaid
flowchart TB
    subgraph "Gang调度流程"
        A["Job提交"]
        B["创建PodGroup"]
        C["等待minAvailable个Pod就绪"]
        D{Pod数 >= minAvailable？}
        E["全部Pod同时调度"]
        F["继续等待"]
        G["任务运行"]
        H["超时失败"]
    end

    A --> B --> C --> D
    D -->|是| E --> G
    D -->|否| F --> C
    F -->|超时| H

    style A fill:#e1f5fe
    style B fill:#fff3e0
    style C fill:#fff3e0
    style D fill:#e3f2fd
    style E fill:#c8e6c9
    style G fill:#c8e6c9
    style H fill:#ffcdd2
```

---

## 7. 结合方案核心知识点

### 7.1 集成架构

```mermaid
graph TB
    subgraph "用户层"
        U["提交训练任务"]
    end

    subgraph "Kueue层"
        KQ["队列管理<br/>配额控制"]
        KWL["Workload管理"]
    end

    subgraph "Volcano层"
        VS["批处理调度<br/>Gang调度"]
        VPG["PodGroup管理"]
    end

    subgraph "Kubernetes层"
        K8S["Pod调度执行"]
    end

    U --> KQ --> VS --> K8S
    U --> KWL --> VPG --> K8S

    style KQ fill:#2196f3
    style KWL fill:#2196f3
    style VS fill:#ff9800
    style VPG fill:#ff9800
```

### 7.2 知识点详解

| 知识点 | 学习分类 | 学习重点 | 解决的问题 |
|--------|----------|----------|------------|
| **集成方式** | 🟢 学会使用 | 配置集成参数 | Kueue + Volcano协作 |
| **调度流程** | 🟠 理解原理 | 两层调度协作 | 队列管理 + 批调度分离 |
| **Workload映射** | 🟠 理解原理 | Workload → Job转换 | 任务抽象转换 |
| **大规模场景** | 🟢 配置使用 | 配额 + Gang结合 | 百节点训练集群 |

---

## 8. 业务场景映射

### 8.1 场景与方案对应

```mermaid
graph LR
    subgraph "业务场景"
        S1["多团队共享GPU集群"]
        S2["大规模分布式训练"]
        S3["混合工作负载调度"]
        S4["高优先级任务抢占"]
        S5["跨节点多GPU任务"]
    end

    subgraph "推荐方案"
        K["Kueue"]
        V["Volcano"]
        KV["Kueue+Volcano"]
    end

    S1 --> K
    S2 --> V
    S3 --> KV
    S4 --> K
    S5 --> V

    style S1 fill:#e3f2fd
    style S2 fill:#fff3e0
    style S3 fill:#c8e6c9
    style S4 fill:#e3f2fd
    style S5 fill:#fff3e0
    style K fill:#2196f3
    style V fill:#ff9800
    style KV fill:#4caf50
```

### 8.2 场景详解

| 场景 | 核心需求 | 推荐方案 | 关键能力 |
|------|----------|----------|----------|
| **多团队共享GPU集群** | 配额隔离、公平分配 | Kueue | ClusterQueue配额、优先级抢占 |
| **大规模分布式训练** | 多GPU协同、同时调度 | Volcano | Gang调度、minAvailable保证 |
| **混合工作负载调度** | 多类型任务统一管理 | Kueue+Volcano | 队列管理 + 批处理调度 |
| **高优先级任务抢占** | 紧急任务快速获取资源 | Kueue | PriorityClass、抢占机制 |
| **跨节点多GPU任务** | All-or-Nothing调度 | Volcano | PodGroup、minMember |

### 8.3 业务场景思考题

---

#### 场景一：多团队共享GPU集群的公平分配

**问题描述：**
公司GPU集群有100张GPU，需要支持3个团队：核心业务团队（需要高优先级）、研究团队（需要公平分配）、临时任务团队（低优先级、可抢占）。

**一句话提示：**
> 用Kueue的ClusterQueue为每个团队设置配额，Cohort实现资源共享，PriorityClass定义优先级层级。

**思考维度：**
- 如何设计配额模型？
- 如何实现团队间资源共享？
- 抢占策略如何配置？

---

#### 场景二：大模型训练的跨节点多GPU调度

**问题描述：**
大模型训练任务需要跨4个节点、每节点2张GPU，共8张GPU协同训练。如何保证调度成功率？

**一句话提示：**
> 用Volcano的Job定义minAvailable=8，Gang调度保证8个Pod同时可用才启动。

**思考维度：**
- Gang调度如何保证All-or-Nothing？
- minAvailable如何设置？
- 如何处理部分节点资源不足？

---

#### 场景三：百节点GPU集群混合工作负载调度

**问题描述：**
百节点GPU集群同时支持：大模型训练（独占多节点）、推理服务（低延迟）、开发调试（交互式）。

**一句话提示：**
> 用Kueue管理队列配额，Volcano处理批调度，分离队列管理和调度执行。

**思考维度：**
- 如何设计统一调度框架？
- 不同任务类型如何分配资源？
- 如何保证服务质量？

---

## 9. 学习路线图

### 9.1 推荐学习顺序

```mermaid
graph TB
    subgraph "阶段一: 基础回顾"
        W1["回顾Week1<br/>资源分配机制"]
    end

    subgraph "阶段二: Kueue学习"
        K1["阅读Kueue概览<br/>组件概念"]
        K2["理解队列模型<br/>两级设计"]
        K3["研读分配算法<br/>scheduler.go"]
        K4["研读抢占机制<br/>preemption.go"]
    end

    subgraph "阶段三: Volcano学习"
        V1["阅读Volcano概览<br/>组件概念"]
        V2["理解Gang调度<br/>PodGroup语义"]
        V3["研读Gang实现<br/>gang.go"]
        V4["研读公平分享<br/>share.go"]
    end

    subgraph "阶段四: 结合方案"
        B1["理解集成架构<br/>两层调度"]
        B2["学习配置方法<br/>集成参数"]
    end

    subgraph "阶段五: 实践应用"
        P1["部署测试环境<br/>实际操作"]
        P2["解决思考题<br/>场景设计"]
    end

    W1 --> K1 --> K2 --> K3 --> K4 --> V1 --> V2 --> V3 --> V4 --> B1 --> B2 --> P1 --> P2

    style W1 fill:#c8e6c9
    style K1 fill:#2196f3
    style K2 fill:#2196f3
    style K3 fill:#ffcdd2
    style K4 fill:#ffcdd2
    style V1 fill:#ff9800
    style V2 fill:#ff9800
    style V3 fill:#ffcdd2
    style V4 fill:#ffcdd2
    style B1 fill:#4caf50
    style B2 fill:#4caf50
    style P1 fill:#9c27b0
    style P2 fill:#9c27b0
```

### 9.2 时间分配建议

| 学习阶段 | 内容 | 预计时间 | 学习方式 |
|----------|------|----------|----------|
| **阶段一** | Week1回顾 | 1小时 | 阅读文档 |
| **阶段二** | Kueue学习 | 6小时 | 文档 + 源码 |
| **阶段三** | Volcano学习 | 6小时 | 文档 + 源码 |
| **阶段四** | 结合方案 | 3小时 | 文档 + 配置 |
| **阶段五** | 实践应用 | 4小时 | 实际部署 |
| **总计** | | **20小时** | |

---

## 附录

### A. 开源项目链接汇总

| 项目 | 链接 | 学习方式 |
|------|------|----------|
| Kueue | https://github.com/kubernetes-sigs/kueue | 🔴 研读源码 |
| Volcano | https://github.com/volcano-sh/volcano | 🔴 研读源码 |
| volcano-kueue-integration | Volcano集成文档 | 🟢 学会使用 |
| MPIJob Operator | https://github.com/kubeflow/mpi-operator | 🟢 学会使用 |
| TFJob Operator | https://github.com/kubeflow/tf-training-operator | 🟢 学会使用 |

### B. 源码研读重点文件

**Kueue：**
- `pkg/scheduler/scheduler.go` - 分配入口
- `pkg/preemption/preemption.go` - 抢占逻辑
- `pkg/queue/cluster_queue.go` - 队列状态
- `apis/kueue/v1beta1/*.go` - CRD定义

**Volcano：**
- `pkg/scheduler/plugins/gang/gang.go` - Gang调度
- `pkg/scheduler/plugins/share/share.go` - 公平分享
- `pkg/scheduler/framework/framework.go` - 调度框架
- `apis/batch/v1alpha1/*.go` - Job CRD

### C. 参考资料

- [Kueue官方文档](https://kueue.sigs.k8s.io/docs/)
- [Volcano官方文档](https://volcano.sh/en/docs/)
- [Kubernetes调度框架](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/)
- [CNCF批量调度白皮书](https://www.cncf.io/blog/2023/11/27/batch-scheduling-in-kubernetes/)

---

> 本学习指导文档明确了知识点分类、开源项目研读指引和分层学习路径，建议按阶段顺序学习，重点研读源码部分。