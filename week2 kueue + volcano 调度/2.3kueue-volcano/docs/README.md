# Kueue + Volcano结合方案详解

> 理解两层调度架构的协作流程，掌握大规模AI训练集群的统一调度方案设计。

---

## 目录

- [1. 概述](#1-概述)
- [2. 集成架构](#2-集成架构)
- [3. 协作流程](#3-协作流程)
- [4. 配置方式](#4-配置方式)
- [5. 业务场景应用](#5-业务场景应用)
- [6. 最佳实践](#6-最佳实践)
- [附录](#附录)

---

## 1. 概述

### 1.1 为什么需要结合方案

```mermaid
graph TB
    subgraph "单一方案局限"
        K["Kueue局限<br/>无Gang调度能力"]
        V["Volcano局限<br/>配额管理较弱"]
    end

    subgraph "结合方案优势"
        A["Kueue: 队列管理"]
        B["Volcano: 批调度"]
        C["互补协作"]
    end

    K --> A --> C
    V --> B --> C

    style K fill:#ffcdd2
    style V fill:#ffcdd2
    style A fill:#2196f3
    style B fill:#ff9800
    style C fill:#4caf50
```

**单一方案局限：**

| 方案 | 局限 | 缺失能力 |
|------|------|----------|
| **Kueue** | 无Gang调度 | 分布式训练同时启动保证 |
| **Volcano** | 配额管理较弱 | 多租户公平分配、抢占 |

### 1.2 结合方案价值

```mermaid
graph TB
    subgraph "结合方案能力"
        C1["配额管理<br/>Kueue ClusterQueue"]
        C2["队列管理<br/>Kueue两级队列"]
        C3["优先级抢占<br/>Kueue抢占机制"]
        C4["Gang调度<br/>Volcano"]
        C5["公平分享<br/>Volcano Share"]
    end

    C1 --> C2 --> C3 --> C4 --> C5

    style C1 fill:#2196f3
    style C2 fill:#2196f3
    style C3 fill:#2196f3
    style C4 fill:#ff9800
    style C5 fill:#ff9800
```

### 1.3 适用场景

| 场景 | 结合方案优势 |
|------|-------------|
| **大规模训练集群** | 配额管理 + Gang调度 |
| **多租户GPU集群** | 公平分配 + 抢占 |
| **混合工作负载** | 统一调度入口 |
| **跨节点分布式训练** | 队列管理 + 同时启动保证 |

---

## 2. 集成架构

### 2.1 两层调度架构

```mermaid
graph TB
    subgraph "用户层"
        U["用户提交任务"]
    end

    subgraph "Kueue层：队列管理"
        K1["Workload接收"]
        K2["配额检查"]
        K3["队列分配"]
        K4["抢占决策"]
    end

    subgraph "Volcano层：调度执行"
        V1["PodGroup创建"]
        V2["Gang调度检查"]
        V3["节点选择"]
        V4["Pod创建"]
    end

    subgraph "Kubernetes层"
        K8S["Pod运行"]
    end

    U --> K1 --> K2 --> K3 --> V1 --> V2 --> V3 --> V4 --> K8S
    K2 -->|配额不足| K4 -->|抢占| K3

    style U fill:#e1f5fe
    style K1 fill:#2196f3
    style K2 fill:#2196f3
    style K3 fill:#2196f3
    style K4 fill:#2196f3
    style V1 fill:#ff9800
    style V2 fill:#ff9800
    style V3 fill:#ff9800
    style V4 fill:#ff9800
    style K8S fill:#4caf50
```

### 2.2 职责分离

```mermaid
graph LR
    subgraph "Kueue职责"
        KQ1["多租户配额管理"]
        KQ2["队列排队分配"]
        KQ3["优先级抢占"]
        KQ4["Workload状态管理"]
    end

    subgraph "Volcano职责"
        VQ1["Gang调度"]
        VQ2["公平分享"]
        VQ3["节点选择"]
        VQ4["Pod创建"]
    end

    KQ1 --> VQ1
    KQ2 --> VQ2
    KQ3 --> VQ3
    KQ4 --> VQ4

    style KQ1 fill:#2196f3
    style KQ2 fill:#2196f3
    style KQ3 fill:#2196f3
    style KQ4 fill:#2196f3
    style VQ1 fill:#ff9800
    style VQ2 fill:#ff9800
    style VQ3 fill:#ff9800
    style VQ4 fill:#ff9800
```

| 层级 | 职责 | 关注点 |
|------|------|--------|
| **Kueue** | 队列管理、配额控制 | 多租户公平分配 |
| **Volcano** | 批调度执行 | 任务调度语义 |

### 2.3 组件交互

```mermaid
sequenceDiagram
    participant User as 用户
    participant Kueue as Kueue
    participant Volcano as Volcano
    participant K8s as Kubernetes

    Note over User,K8s: === 任务提交流程 ===

    User->>Kueue: 提交Volcano Job（带queue annotation）
    Kueue->>Kueue: 创建Workload
    Kueue->>Kueue: 检查ClusterQueue配额

    Note over Kueue: 配额充足？
    Kueue->>Kueue: 分配Workload到ClusterQueue
    Kueue->>Volcano: 通知Volcano接管Job

    Volcano->>Volcano: 创建PodGroup
    Volcano->>Volcano: Gang调度检查

    Note over Volcano: minAvailable满足？
    Volcano->>K8s: 创建Pod
    K8s-->>User: Pod运行

    Note over User,K8s: === 抢占流程 ===

    User->>Kueue: 提交高优先级Job
    Kueue->>Kueue: 配额不足，触发抢占
    Kueue->>K8s: 删除低优先级Pod
    Kueue->>Volcano: 通知接管高优先级Job
    Volcano->>K8s: 创建高优先级Pod
```

---

## 3. 协作流程

### 3.1 任务提交流程

```mermaid
flowchart TB
    subgraph "提交流程"
        A["用户提交Volcano Job"]
        B["Job带kueue annotation"]
        C["Kueue创建Workload"]
        D["Kueue检查配额"]
        E{配额充足？}
        F["分配到ClusterQueue"]
        G["排队等待"]
        H["Kueue通知Volcano"]
        I["Volcano创建PodGroup"]
        J["Volcano执行调度"]
    end

    A --> B --> C --> D --> E
    E -->|是| F --> H --> I --> J
    E -->|否| G --> F

    style A fill:#e1f5fe
    style B fill:#fff3e0
    style C fill:#2196f3
    style D fill:#2196f3
    style E fill:#e3f2fd
    style F fill:#c8e6c9
    style G fill:#fff3e0
    style H fill:#ff9800
    style I fill:#ff9800
    style J fill:#ff9800
```

### 3.2 Gang调度协作

```mermaid
flowchart TB
    subgraph "Gang调度协作"
        A["Kueue分配配额"]
        B["Volcano接管Job"]
        C["创建PodGroup"]
        D["检查minAvailable"]
        E{Pod数 >= minAvailable？}
        F["全部Pod启动"]
        G["继续等待"]
        H["资源预留"]
    end

    A --> B --> C --> D --> E
    E -->|是| F
    E -->|否| G --> H --> D

    style A fill:#2196f3
    style B fill:#ff9800
    style C fill:#ff9800
    style D fill:#ff9800
    style E fill:#e3f2fd
    style F fill:#c8e6c9
    style G fill:#fff3e0
    style H fill:#fff3e0
```

### 3.3 抢占协作

```mermaid
flowchart TB
    subgraph "抢占协作"
        A["高优先级Job提交"]
        B["Kueue检查配额"]
        C{配额充足？}
        D["直接分配"]
        E["触发抢占"]
        F["选择抢占目标"]
        G["删除低优先级Pod"]
        H["释放配额"]
        I["分配高优先级Job"]
        J["Volcano接管"]
    end

    A --> B --> C
    C -->|是| D --> J
    C -->|否| E --> F --> G --> H --> I --> J

    style A fill:#e1f5fe
    style B fill:#2196f3
    style C fill:#e3f2fd
    style D fill:#c8e6c9
    style E fill:#ffcdd2
    style F fill:#2196f3
    style G fill:#ffcdd2
    style H fill:#c8e6c9
    style I fill:#c8e6c9
    style J fill:#ff9800
```

---

## 4. 配置方式

### 4.1 Kueue配置

```yaml
# ============================================================
# ClusterQueue: 配额管理
# ============================================================
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
  name: training-cluster-queue
spec:
  namespaceSelector:
    matchLabels:
      purpose: training

  cohort: gpu-cohort

  resourceGroups:
    - coveredResources: ["nvidia.com/gpu"]
      flavors:
        - name: default-flavor
          resources:
            - name: "nvidia.com/gpu"
              nominalQuota: 50
              borrowingLimit: 20

  preemption:
    reclaimWithinCohort: Any
    withinClusterQueue: LowerPriority
```

### 4.2 Volcano Job配置

```yaml
# ============================================================
# Volcano Job: Kueue接管队列管理
# ============================================================
apiVersion: batch.volcano.sh/v1alpha1
kind: Job
metadata:
  name: distributed-training
  annotations:
    # ============================================================
    # Kueue接管此Job的队列管理
    # ============================================================
    kueue.x-k8s.io/queue-name: training-local-queue
spec:
  # ============================================================
  # Volcano负责Gang调度
  # ============================================================
  minAvailable: 8

  plugins:
    gang:
      minAvailable: 8

  tasks:
    - replicas: 8
      name: worker
      template:
        spec:
          containers:
            - name: pytorch
              resources:
                limits:
                  nvidia.com/gpu: 1
```

### 4.3 LocalQueue配置

```yaml
# ============================================================
# LocalQueue: 命名空间提交入口
# ============================================================
apiVersion: kueue.x-k8s.io/v1beta1
kind: LocalQueue
metadata:
  name: training-local-queue
  namespace: training-ns
spec:
  clusterQueue: training-cluster-queue
```

---

## 5. 业务场景应用

### 5.1 百节点混合工作负载

```mermaid
graph TB
    subgraph "业务场景"
        S1["大模型训练<br/>独占多节点"]
        S2["推理服务<br/>低延迟"]
        S3["开发调试<br/>交互式"]
    end

    subgraph "Kueue队列设计"
        Q1["训练队列<br/>高配额50 GPU"]
        Q2["推理队列<br/>中配额30 GPU"]
        Q3["开发队列<br/>低配额20 GPU"]
    end

    subgraph "Volcano调度"
        V["Gang调度<br/>分布式训练"]
    end

    S1 --> Q1 --> V
    S2 --> Q2
    S3 --> Q3

    style S1 fill:#ffcdd2
    style S2 fill:#fff3e0
    style S3 fill:#c8e6c9
    style Q1 fill:#2196f3
    style Q2 fill:#2196f3
    style Q3 fill:#2196f3
    style V fill:#ff9800
```

**配置方案：**

```yaml
# ============================================================
# 训练队列：高配额，支持Gang调度
# ============================================================
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
  name: training-queue
spec:
  cohort: shared-cohort
  resourceGroups:
    - flavors:
        - resources:
            - name: "nvidia.com/gpu"
              nominalQuota: 50

---
# ============================================================
# 推理队列：中配额，优先级抢占
# ============================================================
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
  name: inference-queue
spec:
  cohort: shared-cohort
  resourceGroups:
    - flavors:
        - resources:
            - name: "nvidia.com/gpu"
              nominalQuota: 30

  preemption:
    reclaimWithinCohort: LowerPriority

---
# ============================================================
# 开发队列：低配额，可被抢占
# ============================================================
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
  name: dev-queue
spec:
  cohort: shared-cohort
  resourceGroups:
    - flavors:
        - resources:
            - name: "nvidia.com/gpu"
              nominalQuota: 20

  preemption:
    reclaimWithinCohort: Never
```

### 5.2 分布式训练场景

```mermaid
graph TB
    subgraph "场景：8节点64GPU训练"
        N1["节点1-8<br/>每节点8 GPU"]
        T["训练任务<br/>需要64 GPU"]
    end

    subgraph "方案"
        K["Kueue分配配额"]
        V["Volcano Gang调度"]
        P["64 Worker Pod"]
    end

    T --> K --> V --> P --> N1

    style N1 fill:#fff3e0
    style T fill:#e1f5fe
    style K fill:#2196f3
    style V fill:#ff9800
    style P fill:#c8e6c9
```

**配置方案：**

```yaml
# ============================================================
# 8节点64GPU分布式训练
# ============================================================
apiVersion: batch.volcano.sh/v1alpha1
kind: Job
metadata:
  name: large-scale-training
  annotations:
    kueue.x-k8s.io/queue-name: training-queue
spec:
  minAvailable: 64           # Gang调度：必须64个Pod同时可用
  tasks:
    - replicas: 64
      name: worker
      template:
        spec:
          containers:
            - name: pytorch
              resources:
                limits:
                  nvidia.com/gpu: 1    # 每个Pod1张GPU
```

### 5.3 多租户场景

```mermaid
graph TB
    subgraph "多租户设计"
        T1["团队A: 核心业务"]
        T2["团队B: 研究"]
        T3["团队C: 开发"]
    end

    subgraph "Kueue队列"
        Q1["Queue A: 30 GPU, 可抢占"]
        Q2["Queue B: 40 GPU, 中优先级"]
        Q3["Queue C: 20 GPU, 低优先级"]
    end

    subgraph "共享池"
        C["Cohort<br/>可借用资源"]
    end

    T1 --> Q1 --> C
    T2 --> Q2 --> C
    T3 --> Q3 --> C

    style T1 fill:#ffcdd2
    style T2 fill:#fff3e0
    style T3 fill:#c8e6c9
    style Q1 fill:#2196f3
    style Q2 fill:#2196f3
    style Q3 fill:#2196f3
    style C fill:#9e9e9e
```

---

## 6. 最佳实践

### 6.1 配额设计建议

```mermaid
graph TB
    subgraph "配额设计原则"
        P1["核心业务: 高配额 + 抢占"]
        P2["研究团队: 中配额 + 借用"]
        P3["开发调试: 低配额 + 可被抢占"]
    end

    P1 --> P2 --> P3

    style P1 fill:#ffcdd2
    style P2 fill:#fff3e0
    style P3 fill:#c8e6c9
```

| 团队类型 | 配额策略 | 抢占策略 |
|----------|----------|----------|
| **核心业务** | nominalQuota ≥ 日常需求 | reclaimWithinCohort: Any |
| **研究团队** | nominalQuota + borrowingLimit | reclaimWithinCohort: LowerPriority |
| **开发调试** | 仅borrowingLimit | reclaimWithinCohort: Never |

### 6.2 Gang调度配置建议

| 任务类型 | minAvailable设置 | 说明 |
|----------|------------------|------|
| **严格分布式训练** | = 总Pod数 | 全部节点必须可用 |
| **弹性训练** | < 总Pod数 | 允许部分节点故障 |
| **单节点多GPU** | = 1 | 单节点即可 |

### 6.3 常见问题排查

| 问题 | 可能原因 | 解决方式 |
|------|----------|----------|
| Job一直Pending | Kueue配额不足 | 检查ClusterQueue配额 |
| Gang调度等待 | minAvailable不满足 | 检查资源是否足够 |
| Volcano未接管 | annotation缺失 | 添加kueue annotation |
| 抢占未触发 | preemption策略 | 检查preemption配置 |

---

## 附录

### A. 集成配置清单

| 配置项 | 来源 | 说明 |
|--------|------|------|
| ClusterQueue | Kueue | 团队配额 |
| LocalQueue | Kueue | 提交入口 |
| Volcano Job | Volcano | 批处理任务 |
| kueue annotation | Volcano Job | 关联LocalQueue |

### B. 参考资料

- [Kueue官方文档](https://kueue.sigs.k8s.io/docs/)
- [Volcano官方文档](https://volcano.sh/en/docs/)
- [Volcano-Kueue集成](https://volcano.sh/en/docs/integration/kueue/)

---

> 本文档详解了Kueue+Volcano结合方案的两层调度架构和协作流程，建议结合业务场景设计具体配置方案。