# Volcano详解：批处理调度与Gang Scheduling

> 深入理解 Volcano 的Gang调度、公平分享算法、调度框架，掌握分布式训练调度方案。

---

## 目录

- [1. 概述](#1-概述)
- [2. 核心组件](#2-核心组件)
- [3. Gang调度详解](#3-gang调度详解)
- [4. 公平分享算法](#4-公平分享算法)
- [5. 调度框架](#5-调度框架)
- [6. 任务依赖](#6-任务依赖)
- [7. 业务场景应用](#7-业务场景应用)
- [8. 最佳实践](#8-最佳实践)
- [附录](#附录)

---

## 1. 概述

### 1.1 什么是Volcano

Volcano是Kubernetes批处理任务调度系统，专注于解决分布式训练和批处理任务的调度问题。

```mermaid
graph TB
    subgraph "Volcano定位"
        Q1["核心问题<br/>分布式任务如何协同调度？"]
        Q2["解决方案<br/>Gang调度 + 公平分享 + 任务依赖"]
    end

    subgraph "核心能力"
        C1["Gang调度<br/>All-or-Nothing"]
        C2["公平分享<br/>DRF算法"]
        C3["任务依赖<br/>DAG链"]
    end

    Q1 --> Q2 --> C1 --> C2 --> C3

    style Q1 fill:#ffcdd2
    style Q2 fill:#c8e6c9
    style C1 fill:#ff9800
    style C2 fill:#ff9800
    style C3 fill:#ff9800
```

### 1.2 核心特性

| 特性 | 说明 | 解决的问题 |
|------|------|------------|
| **Gang调度** | All-or-Nothing调度语义 | 分布式训练同时启动 |
| **公平分享** | DRF/SSJ公平分配算法 | 多任务公平使用资源 |
| **任务依赖** | DAG依赖链 | 任务间依赖关系 |
| **批处理Job** | 统一的Job CRD | 批处理任务抽象 |
| **调度框架** | 插件化扩展 | 自定义调度策略 |

### 1.3 与Kueue的关系

```mermaid
graph LR
    subgraph "职责分离"
        K["Kueue<br/>队列管理层"]
        V["Volcano<br/>调度执行层"]
    end

    K -->|任务排队| V -->|批调度执行| K8S["Kubernetes"]

    style K fill:#2196f3
    style V fill:#ff9800
    style K8S fill:#4caf50
```

| 维度 | Kueue | Volcano |
|------|-------|---------|
| **职责** | 队列管理、配额控制 | 批调度执行 |
| **关注点** | 多租户公平分配 | 任务调度语义 |
| **场景** | 任务排队等待 | 分布式任务调度 |

---

## 2. 核心组件

### 2.1 组件总览

```mermaid
graph TB
    subgraph "用户层"
        U["用户提交任务"]
    end

    subgraph "Volcano组件"
        Q["Queue<br/>队列"]
        J["Job<br/>批处理任务"]
        PG["PodGroup<br/>调度组"]
    end

    subgraph "调度插件"
        GANG["Gang"]
        SHARE["Share"]
        PRI["Priority"]
        BINP["Binpack"]
    end

    subgraph "Kubernetes"
        K8S["Pod执行"]
    end

    U --> J --> PG --> GANG --> K8S
    PG --> SHARE --> K8S
    PG --> PRI --> K8S
    PG --> BINP --> K8S
    J --> Q

    style U fill:#e1f5fe
    style Q fill:#ff9800
    style J fill:#ff9800
    style PG fill:#ff9800
    style GANG fill:#ffcdd2
    style SHARE fill:#ffcdd2
    style PRI fill:#fff3e0
    style BINP fill:#fff3e0
    style K8S fill:#4caf50
```

### 2.2 组件详解

#### Queue（队列）

```mermaid
graph TB
    subgraph "Queue结构"
        Q1["权重<br/>weight"]
        Q2["能力<br/>capability"]
        Q3["状态<br/>state"]
    end

    Q1 --> Q2 --> Q3

    style Q1 fill:#fff3e0
    style Q2 fill:#fff3e0
    style Q3 fill:#fff3e0
```

| 字段 | 说明 | 示例 |
|------|------|------|
| `spec.weight` | 队列权重 | 3（3倍资源分配） |
| `spec.capability` | 资源能力 | "nvidia.com/gpu: 10" |
| `spec.state` | 队列状态 | Open/Closed |

#### Job（批处理任务）

```mermaid
graph TB
    subgraph "Job结构"
        J1["minAvailable<br/>Gang调度参数"]
        J2["tasks<br/>任务模板"]
        J3["queue<br/>所属队列"]
        J4["plugins<br/>调度插件"]
    end

    J1 --> J2 --> J3 --> J4

    style J1 fill:#ffcdd2
    style J2 fill:#fff3e0
    style J3 fill:#2196f3
    style J4 fill:#ff9800
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `spec.minAvailable` | 最少可用Pod数 | Gang调度核心参数 |
| `spec.tasks` | 任务模板列表 | 定义Worker/Master |
| `spec.queue` | 所属队列 | 资源分配依据 |
| `spec.plugins` | 调度插件配置 | 启用Gang/Share等 |

#### PodGroup（调度组）

```mermaid
graph TB
    subgraph "PodGroup结构"
        P1["minMember<br/>最小成员数"]
        P2["queue<br/>所属队列"]
        P3["任务关联<br/>Job关联"]
    end

    P1 --> P2 --> P3

    style P1 fill:#ffcdd2
    style P2 fill:#2196f3
    style P3 fill:#fff3e0
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `spec.minMember` | 最小成员数 | Gang调度判断条件 |
| `spec.queue` | 所属队列 | 路由到Queue |

---

## 3. Gang调度详解

### 3.1 Gang调度原理

```mermaid
graph TB
    subgraph "Gang调度语义"
        G1["All-or-Nothing<br/>要么全部调度，要么全部不调度"]
        G2["分布式训练需求<br/>Worker需要同时启动通信"]
        G3["避免资源浪费<br/>部分Pod启动等待其他"]
    end

    G1 --> G2 --> G3

    style G1 fill:#ff9800
    style G2 fill:#fff3e0
    style G3 fill:#c8e6c9
```

**为什么需要Gang调度：**

| 问题 | Gang调度解决 |
|------|-------------|
| 分布式训练Worker通信依赖 | 保证全部Worker同时启动 |
| 部分Pod启动等待其他Pod | 等待全部可用才启动 |
| 资源碎片导致任务失败 | 预留资源避免碎片 |

### 3.2 Gang调度流程

```mermaid
flowchart TB
    subgraph "Gang调度流程"
        A["Job提交"]
        B["创建PodGroup"]
        C["创建Pod"]
        D["检查PodGroup状态"]
        E{运行Pod数 >= minAvailable？}
        F["全部Pod启动"]
        G["继续等待"]
        H["超时检查"]
        I["任务失败"]
    end

    A --> B --> C --> D --> E
    E -->|是| F
    E -->|否| G --> D
    G -->|超时| H --> I

    style A fill:#e1f5fe
    style B fill:#fff3e0
    style C fill:#fff3e0
    style E fill:#e3f2fd
    style F fill:#c8e6c9
    style G fill:#fff3e0
    style I fill:#ffcdd2
```

### 3.3 minAvailable设置

```mermaid
graph TB
    subgraph "minAvailable设置策略"
        S1["等于总Pod数<br/>严格All-or-Nothing"]
        S2["小于总Pod数<br/>允许部分容忍"]
        S3["根据任务需求<br/>最小可用数"]
    end

    S1 --> S2 --> S3

    style S1 fill:#ffcdd2
    style S2 fill:#fff3e0
    style S3 fill:#c8e6c9
```

| 设置策略 | 示例 | 适用场景 |
|----------|------|----------|
| **等于总Pod数** | minAvailable=8, replicas=8 | 严格All-or-Nothing |
| **小于总Pod数** | minAvailable=6, replicas=8 | 允许部分节点故障 |
| **根据需求** | 训练需要最小Worker数 | 弹性训练 |

### 3.4 源码研读：gang.go

```go
// ============================================================
// Volcano Gang调度插件核心逻辑
// ============================================================
// 文件位置: pkg/scheduler/plugins/gang/gang.go
// 学习重点: 理解All-or-Nothing调度实现

func (g *Gang) OnSessionOpen(ssn *Session) {
    // === 步骤1: 注册PodGroup检查函数 ===
    ssn.AddPredicateFn(g.Name(), func(pod *v1.Pod) bool {
        pg := ssn.GetPodGroup(pod)

        // === 步骤2: 检查minMember条件 ===
        // 计算当前可调度Pod数
        readyPods := ssn.GetReadyPods(pg)

        // 如果已有足够Pod，继续调度
        if readyPods >= pg.Spec.MinMember {
            return true
        }

        // === 歐骤3: 检查资源是否足够调度全部 ===
        // 预留资源，避免碎片
        requiredResources := pg.GetTotalResources()
        availableResources := ssn.GetAvailableResources()

        if availableResources >= requiredResources {
            // 资源足够，允许调度
            return true
        }

        // 资源不足，拒绝调度
        return false
    })
}
```

**源码研读重点：**

| 函数/方法 | 研读内容 | 理解目标 |
|-----------|----------|----------|
| `OnSessionOpen()` | 插件初始化 | 理解插件注册机制 |
| `PredicateFn()` | 过滤函数 | 理解调度条件判断 |
| `GetReadyPods()` | 统计已调度Pod | 理解状态计算 |
| `GetAvailableResources()` | 计算可用资源 | 理解资源预留 |

---

## 4. 公平分享算法

### 4.1 DRF算法原理

```mermaid
graph TB
    subgraph "DRF (Dominant Resource Fairness)"
        D1["核心思想<br/>按主导资源比例公平分配"]
        D2["主导资源<br/>任务消耗最多的资源类型"]
        D3["公平分配<br/>使各任务主导资源比例相等"]
    end

    D1 --> D2 --> D3

    style D1 fill:#ff9800
    style D2 fill:#fff3e0
    style D3 fill:#c8e6c9
```

**DRF计算示例：**

| 任务 | GPU请求 | CPU请求 | 主导资源 | 主导比例 |
|------|---------|---------|----------|----------|
| 任务A | 4 GPU | 8 CPU | GPU | 4/100 = 4% |
| 任务B | 2 GPU | 32 CPU | CPU | 32/200 = 16% |
| 任务C | 8 GPU | 4 CPU | GPU | 8/100 = 8% |

**公平分配：** 使任务A、B、C的主导资源比例相等。

### 4.2 SSJ算法原理

```mermaid
graph TB
    subgraph "SSJ (Slot Share with Jobs)"
        S1["核心思想<br/>按队列权重和任务数分配"]
        S2["权重因素<br/>Queue.spec.weight"]
        S3["任务数因素<br/>队列内任务数量"]
    end

    S1 --> S2 --> S3

    style S1 fill:#ff9800
    style S2 fill:#fff3e0
    style S3 fill:#c8e6c9
```

**SSJ计算公式：**

```
队列分配比例 = weight / (weight × jobCount)

其中:
- weight: Queue权重
- jobCount: 队列内任务数量
```

### 4.3 公平分享流程

```mermaid
flowchart TB
    subgraph "公平分享流程"
        A["多任务竞争资源"]
        B["计算各任务资源需求"]
        C["确定主导资源"]
        D["计算公平分配比例"]
        E["按比例分配资源"]
    end

    A --> B --> C --> D --> E

    style A fill:#e1f5fe
    style B fill:#fff3e0
    style C fill:#e3f2fd
    style D fill:#ff9800
    style E fill:#c8e6c9
```

### 4.4 源码研读：share.go

```go
// ============================================================
// Volcano公平分享插件核心逻辑
// ============================================================
// 文件位置: pkg/scheduler/plugins/share/share.go
// 学习重点: 理解DRF/SSJ算法实现

func (s *Share) OnSessionOpen(ssn *Session) {
    // === 步骤1: 注册打分函数 ===
    ssn.AddScoreFn(g.Name(), func(pod *v1.Pod) int64 {
        pg := ssn.GetPodGroup(pod)
        queue := ssn.GetQueue(pg.Spec.Queue)

        // === 步骤2: 计算队列当前主导资源比例 ===
        dominantShare := s.calculateDominantShare(queue)

        // === 歧骤3: 主导比例低得高分 ===
        // 反比例打分：比例越低，分数越高
        score := maxScore - dominantShare * scoreFactor

        return score
    })
}

func (s *Share) calculateDominantShare(queue *Queue) float64 {
    // === 计算各资源使用比例 ===
    gpuShare := queue.UsedGPU / queue.TotalGPU
    cpuShare := queue.UsedCPU / queue.TotalCPU
    memShare := queue.UsedMemory / queue.TotalMemory

    // === 返回最大比例（主导资源） ===
    return max(gpuShare, cpuShare, memShare)
}
```

---

## 5. 调度框架

### 5.1 插件扩展点

```mermaid
graph TB
    subgraph "Volcano调度框架扩展点"
        E1["Predicate<br/>过滤插件"]
        E2["Score<br/>打分插件"]
        E3["Preempt<br/>抢占插件"]
        E4["Reclaim<br/>回收插件"]
    end

    E1 --> E2 --> E3 --> E4

    style E1 fill:#ffcdd2
    style E2 fill:#fff3e0
    style E3 fill:#2196f3
    style E4 fill:#c8e6c9
```

| 扩展点 | 作用 | 内置插件 |
|--------|------|----------|
| **Predicate** | 过滤不满足条件的节点 | Gang、NodePorts |
| **Score** | 为节点打分排序 | Share、Binpack |
| **Preempt** | 决定抢占目标 | Priority |
| **Reclaim** | 决定资源回收 | Share |

### 5.2 插件注册机制

```go
// ============================================================
// Volcano插件注册机制
// ============================================================
// 文件位置: pkg/scheduler/framework/framework.go

type Plugin interface {
    // 插件名称
    Name() string

    // Session开始时注册扩展点
    OnSessionOpen(ssn *Session)

    // Session结束时清理
    OnSessionClose(ssn *Session)
}

// ============================================================
// Session扩展点注册
// ============================================================
type Session struct {
    // Predicate函数列表
    predicates map[string]PredicateFn

    // Score函数列表
    scoreFn    map[string]ScoreFn

    // 抢占函数列表
    preemptFn  map[string]PreemptFn
}
```

### 5.3 自定义插件示例

```go
// ============================================================
// 示例: 自定义GPU拓扑感知调度插件
// ============================================================
package plugins

type GPUTopologyPlugin struct {}

func (p *GPUTopologyPlugin) Name() string {
    return "gpu-topology"
}

func (p *GPUTopologyPlugin) OnSessionOpen(ssn *Session) {
    // 注册打分函数
    ssn.AddScoreFn(p.Name(), func(pod *v1.Pod, node *v1.Node) int64 {
        // 检查GPU NUMA拓扑
        // 优先调度到NUMA亲和的节点
        return p.calculateTopologyScore(pod, node)
    })
}
```

---

## 6. 任务依赖

### 6.1 DAG依赖链

```mermaid
graph TB
    subgraph "任务依赖关系"
        T1["数据准备"]
        T2["模型训练"]
        T3["模型评估"]
        T4["模型部署"]
    end

    T1 --> T2 --> T3 --> T4

    style T1 fill:#c8e6c9
    style T2 fill:#fff3e0
    style T3 fill:#fff3e0
    style T4 fill:#ff9800
```

### 6.2 依赖配置示例

```yaml
# ============================================================
# 任务依赖配置示例
# ============================================================
apiVersion: batch.volcano.sh/v1alpha1
kind: Job
metadata:
  name: training-job
spec:
  depends:
    - name: data-prep-job      # 依赖数据准备任务
      status: Complete         # 状态必须完成
```

---

## 7. 业务场景应用

### 7.1 分布式训练场景

```mermaid
graph TB
    subgraph "场景: 4节点分布式训练"
        N1["节点1: 2张GPU"]
        N2["节点2: 2张GPU"]
        N3["节点3: 2张GPU"]
        N4["节点4: 2张GPU"]
    end

    subgraph "Volcano方案"
        V["Volcano Job"]
        M["minAvailable=8"]
        P["8个Worker Pod"]
    end

    V --> M --> P --> N1 & N2 & N3 & N4

    style N1 fill:#fff3e0
    style N2 fill:#fff3e0
    style N3 fill:#fff3e0
    style N4 fill:#fff3e0
    style V fill:#ff9800
    style M fill:#ffcdd2
    style P fill:#c8e6c9
```

**配置要点：**

```yaml
# ============================================================
# 4节点8GPU分布式训练配置
# ============================================================
apiVersion: batch.volcano.sh/v1alpha1
kind: Job
metadata:
  name: distributed-training
spec:
  minAvailable: 8               # Gang调度：必须8个Pod同时可用
  queue: training-queue
  tasks:
    - replicas: 8
      name: worker
      template:
        spec:
          containers:
            - name: pytorch
              resources:
                limits:
                  nvidia.com/gpu: 1    # 每个Pod1张GPU
```

### 7.2 MPI分布式训练

```yaml
# ============================================================
# MPI分布式训练示例
# ============================================================
apiVersion: batch.volcano.sh/v1alpha1
kind: MPIJob
metadata:
  name: mpi-training
spec:
  minAvailable: 4               # 4个Worker必须同时可用
  queue: mpi-queue
  mpiReplicaSpecs:
    Launcher:
      replicas: 1
      template:
        spec:
          containers:
            - name: mpi-launcher
              image: mpi-image
    Worker:
      replicas: 4
      template:
        spec:
          containers:
            - name: mpi-worker
              resources:
                limits:
                  nvidia.com/gpu: 2
```

---

## 8. 最佳实践

### 8.1 minAvailable设置建议

| 任务类型 | minAvailable设置 | 原因 |
|----------|------------------|------|
| **严格分布式训练** | = replicas | 全部节点必须可用 |
| **弹性训练** | < replicas | 允许部分节点故障 |
| **单节点多GPU** | = 1 | 单节点即可 |

### 8.2 Queue配置建议

```mermaid
graph TB
    subgraph "Queue配置策略"
        Q1["训练队列: 高权重<br/>保证训练资源"]
        Q2["推理队列: 中权重<br/>保证响应速度"]
        Q3["开发队列: 低权重<br/>弹性分配"]
    end

    Q1 --> Q2 --> Q3

    style Q1 fill:#ffcdd2
    style Q2 fill:#fff3e0
    style Q3 fill:#c8e6c9
```

### 8.3 常见问题排查

| 问题 | 可能原因 | 解决方式 |
|------|----------|----------|
| Job一直Pending | minAvailable条件不满足 | 检查资源是否足够 |
| 部分Pod启动后等待 | Gang调度等待其他Pod | 正常现象，等待全部就绪 |
| 任务调度失败 | 资源碎片 | 检查minAvailable和资源总量 |
| 公平分享不生效 | 插件未启用 | 配置plugins启用share |

---

## 附录

### A. CRD字段速查

**Job核心字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `spec.minAvailable` | int32 | Gang调度最小可用数 |
| `spec.tasks[].replicas` | int32 | 任务副本数 |
| `spec.queue` | string | 所属队列 |
| `spec.plugins` | []PluginConfig | 调度插件配置 |

**Queue核心字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `spec.weight` | int64 | 队列权重 |
| `spec.capability` | string | 资源能力 |
| `spec.state` | QueueState | 队列状态 |

### B. 源码研读清单

| 文件 | 研读重点 | 预计时间 |
|------|----------|----------|
| `pkg/scheduler/plugins/gang/gang.go` | Gang调度实现 | 2小时 |
| `pkg/scheduler/plugins/share/share.go` | 公平分享算法 | 2小时 |
| `pkg/scheduler/framework/framework.go` | 调度框架 | 1小时 |
| `apis/batch/v1alpha1/*.go` | Job CRD定义 | 1小时 |

### C. 参考资料

- [Volcano官方文档](https://volcano.sh/en/docs/)
- [Volcano源码](https://github.com/volcano-sh/volcano)
- [Volcano设计文档](https://volcano.sh/en/docs/design/)

---

> 本文档详解了Volcano的Gang调度、公平分享算法和调度框架，建议结合源码研读深入理解实现细节。