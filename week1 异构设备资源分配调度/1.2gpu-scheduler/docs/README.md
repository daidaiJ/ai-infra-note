# Kubernetes GPU 感知调度器

> 一个自定义 Kubernetes 调度器，支持 **GPU 利用率均衡调度** 和 **设备 UUID 精确绑定** 两大核心特性。

---

## 目录

- [1. 概述](#1-概述)
- [2. Kubernetes 调度器架构](#2-kubernetes-调度器架构)
- [3. 调度框架扩展机制](#3-调度框架扩展机制)
- [4. 项目结构](#4-项目结构)
- [5. 核心插件实现](#5-核心插件实现)
- [6. 调度流程详解](#6-调度流程详解)
- [7. 配置与部署](#7-配置与部署)
- [8. 使用示例](#8-使用示例)

---

## 1. 概述

### 1.1 功能特性

| 特性 | 描述 | 适用场景 |
|------|------|----------|
| **GPU 利用率均衡** | 优先调度到低负载 GPU 节点 | 多任务负载均衡、集群资源优化 |
| **设备 UUID 绑定** | 将 Pod 调度到指定 UUID 的 GPU 设备 | 模型调试、硬件亲和性、故障隔离 |

### 1.2 调度决策流程

```mermaid
flowchart TD
    A[Pod 创建] --> B{Pod 指定 GPU UUID?}
    
    B -->|是| C[Filter: 只保留目标设备所在节点]
    B -->|否| D[Filter: 检查 GPU 健康/显存]
    
    C --> E[Score: 目标节点得最高分]
    D --> F[Score: 低利用率节点得高分]
    
    E --> G[选择最高分节点]
    F --> G
    
    G --> H[Bind: 绑定 Pod 到节点]
    H --> I[调度完成]
    
    style A fill:#e1f5fe
    style I fill:#c8e6c9
    style C fill:#fff3e0
    style F fill:#f3e5f5
```

---

## 2. Kubernetes 调度器架构

### 2.1 整体架构

```mermaid
graph TB
    subgraph "Kubernetes 集群"
        API[API Server]
        ETCD[(etcd)]
        
        subgraph "调度器进程"
            INFORMER[Informer<br/>事件监听]
            QUEUE[优先队列<br/>Pod 排队]
            
            subgraph "调度周期"
                PF[PreFilter<br/>预过滤]
                F[Filter<br/>节点过滤]
                PS[PreScore<br/>预打分]
                S[Score<br/>节点打分]
                NS[NormalizeScore<br/>分数归一化]
            end
            
            subgraph "绑定周期"
                RES[Reserve<br/>资源预留]
                PER[Permit<br/>许可检查]
                PRE[PreBind<br/>绑定前处理]
                B[Bind<br/>执行绑定]
                POST[PostBind<br/>绑定后处理]
            end
        end
        
        NODE1[Node 1<br/>GPU: 0%, 显存: 8GB]
        NODE2[Node 2<br/>GPU: 85%, 显存: 2GB]
        NODE3[Node 3<br/>GPU: 45%, 显存: 5GB]
    end
    
    API --> INFORMER
    INFORMER --> QUEUE
    QUEUE --> PF
    PF --> F --> PS --> S --> NS
    NS --> RES --> PER --> PRE --> B --> POST
    B --> API
    API --> ETCD
    F -.-> NODE1
    F -.-> NODE2
    F -.-> NODE3
    S -.-> NODE1
    S -.-> NODE2
    S -.-> NODE3
```

### 2.2 调度周期状态机

```mermaid
stateDiagram-v2
    [*] --> Queued: Pod 入队
    Queued --> PreFilter: 开始调度
    PreFilter --> Filter: 预处理完成
    
    state Filter {
        [*] --> F1: NodeUnschedulable
        F1 --> F2: TaintToleration
        F2 --> F3: NodeAffinity
        F3 --> F4: GPUUtilization
        F4 --> F5: DeviceUUID
        F5 --> [*]: 过滤完成
    }
    
    Filter --> NoNode: 无可用节点
    Filter --> Score: 有候选节点
    
    state Score {
        [*] --> S1: DeviceUUIDMatcher
        S1 --> S2: GPUUtilizationBalancer
        S2 --> S3: NodeResourcesFit
        S3 --> [*]: 打分完成
    }
    
    Score --> Bind: 选择最优节点
    Bind --> [*]: 绑定成功
    
    NoNode --> [*]: 调度失败
```

---

## 3. 调度框架扩展机制

### 3.1 扩展点概览

```mermaid
graph LR
    subgraph "调度周期 Scheduling Cycle"
        QS[QueueSort<br/>队列排序]
        PreF[PreFilter<br/>预过滤]
        Fil[Filter<br/>过滤]
        PostF[PostFilter<br/>过滤后处理]
        PreS[PreScore<br/>预打分]
        Sco[Score<br/>打分]
        NorS[NormalizeScore<br/>归一化]
        Res[Reserve<br/>预留]
        Per[Permit<br/>许可]
    end
    
    subgraph "绑定周期 Binding Cycle"
        PreB[PreBind<br/>绑定前]
        Bin[Bind<br/>绑定]
        PosB[PostBind<br/>绑定后]
    end
    
    QS --> PreF --> Fil --> PostF
    PostF --> PreS --> Sco --> NorS
    NorS --> Res --> Per
    Per --> PreB --> Bin --> PosB
```

### 3.2 各扩展点作用

| 扩展点 | 阶段 | 作用 | 本项目使用 |
|--------|------|------|------------|
| **QueueSort** | 排队 | 决定 Pod 调度优先级 | ❌ 使用默认 |
| **PreFilter** | 预处理 | 准备调度所需数据 | ❌ 未使用 |
| **Filter** | 过滤 | 排除不满足条件的节点 | ✅ 核心功能 |
| **PostFilter** | 后处理 | 处理过滤失败（如抢占） | ❌ 未使用 |
| **PreScore** | 预打分 | 准备打分数据 | ❌ 未使用 |
| **Score** | 打分 | 为节点评分 | ✅ 核心功能 |
| **NormalizeScore** | 归一化 | 统一分数范围 | ❌ 未使用 |
| **Reserve** | 预留 | 预留资源 | ⚠️ 可选 |
| **Permit** | 许可 | 控制调度等待/拒绝 | ❌ 未使用 |
| **PreBind** | 绑定前 | 绑定前验证/准备 | ⚠️ 可选 |
| **Bind** | 绑定 | 执行 Pod-Node 绑定 | ❌ 使用默认 |
| **PostBind** | 绑定后 | 绑定后清理/通知 | ❌ 未使用 |

### 3.3 插件接口定义

```go
// 所有插件必须实现的基础接口
type Plugin interface {
    Name() string
}

// Filter 插件接口
type FilterPlugin interface {
    Plugin
    Filter(ctx context.Context, state *CycleState, pod *v1.Pod, nodeInfo *NodeInfo) *Status
}

// Score 插件接口
type ScorePlugin interface {
    Plugin
    Score(ctx context.Context, state *CycleState, pod *v1.Pod, nodeName string) (int64, *Status)
    ScoreExtensions() ScoreExtensions  // 可选
}
```

---

## 4. 项目结构

```
gpu-scheduler/
├── cmd/
│   └── scheduler/
│       └── main.go              # 入口：注册插件、启动调度器
├── pkg/
│   ├── gpuinfo/
│   │   └── gpuinfo.go           # Mock: GPU 信息获取接口
│   └── plugins/
│       ├── gpuutilization/
│       │   └── gpuutilization.go # 插件: GPU 利用率均衡
│       └── deviceuuid/
│           └── deviceuuid.go    # 插件: 设备 UUID 匹配
├── config/
│   └── scheduler-config.yaml     # 调度器配置文件
├── manifests/
│   └── deployment.yaml           # K8s 部署清单
└── docs/
    └── README.md                 # 本文档
```

---

## 5. 核心插件实现

### 5.1 GPU 利用率均衡插件

#### 5.1.1 设计目标

```mermaid
graph LR
    subgraph "调度前"
        N1A["Node-A<br/>GPU: 20%"]
        N2A["Node-B<br/>GPU: 80%"]
        N3A["Node-C<br/>GPU: 50%"]
    end
    
    subgraph "调度决策"
        SCORE[评分计算<br/>低利用率 = 高分]
    end
    
    subgraph "调度后"
        N1B["Node-A<br/>GPU: 35%<br/>✅ 被选中"]
        N2B["Node-B<br/>GPU: 80%"]
        N3B["Node-C<br/>GPU: 50%"]
    end
    
    N1A --> SCORE
    N2A --> SCORE
    N3A --> SCORE
    SCORE --> N1B
    SCORE -.-> N2B
    SCORE -.-> N3B
    
    style N1A fill:#c8e6c9
    style N1B fill:#c8e6c9
    style SCORE fill:#fff9c4
```

#### 5.1.2 核心逻辑

```go
// Filter: 过滤不健康的 GPU 节点
func (p *GPUUtilizationPlugin) Filter(...) *framework.Status {
    // 1. 检查节点是否有 GPU 资源
    if !p.hasGPUResources(nodeInfo) {
        return Success  // 无 GPU 节点交给其他插件处理
    }
    
    // 2. 检查 GPU 健康状态
    if !p.isGPUHealthy(nodeName) {
        return Unschedulable  // 过滤掉不健康节点
    }
    
    // 3. 检查显存是否充足
    if requestedMemory > availableMemory {
        return Unschedulable  // 过滤掉显存不足节点
    }
    
    return Success
}

// Score: 利用率低 → 分数高
func (p *GPUUtilizationPlugin) Score(...) (int64, *framework.Status) {
    utilization := gpuinfo.GetNodeGPUUtilization(nodeName)
    score := 100 - utilization  // 核心公式
    return score, Success
}
```

#### 5.1.3 评分算法

```mermaid
graph LR
    U["GPU 利用率 (%)"] --> F["Score = 100 - 利用率"]
    
    F --> S1["0% → Score: 100"]
    F --> S2["25% → Score: 75"]
    F --> S3["50% → Score: 50"]
    F --> S4["75% → Score: 25"]
    F --> S5["100% → Score: 0"]
    
    style S1 fill:#4caf50
    style S2 fill:#8bc34a
    style S3 fill:#ffeb3b
    style S4 fill:#ff9800
    style S5 fill:#f44336
```

### 5.2 设备 UUID 匹配插件

#### 5.2.1 设计目标

```mermaid
sequenceDiagram
    participant User as 用户
    participant API as API Server
    participant Sched as GPU 调度器
    participant N1 as Node-A<br/>GPU-001, GPU-002
    participant N2 as Node-B<br/>GPU-003, GPU-004
    
    User->>API: 创建 Pod<br/>annotation: target-gpu-uuid=GPU-003
    API->>Sched: Pod 待调度事件
    Sched->>Sched: Filter: 检查 GPU-003 所在节点
    Note over Sched: GPU-003 → Node-B
    Schem->>N1: Filter: 不包含 GPU-003
    Schem->>N2: Filter: 包含 GPU-003 ✓
    Sched->>Sched: Score: Node-B = 100, Node-A = 0
    Sched->>API: Bind Pod → Node-B
    API->>N2: Pod 调度到 Node-B
```

#### 5.2.2 核心逻辑

```go
// Filter: 只保留目标设备所在节点
func (p *DeviceUUIDPlugin) Filter(...) *framework.Status {
    targetUUIDs := p.getPodTargetUUIDs(pod)
    if len(targetUUIDs) == 0 {
        return Success  // 未指定 UUID，交给其他插件
    }
    
    for _, uuid := range targetUUIDs {
        deviceNode := gpuinfo.GetNodeByDeviceUUID(uuid)
        if deviceNode == nodeName {
            return Success  // 找到目标设备
        }
    }
    
    return Unschedulable  // 节点不包含目标设备
}

// Score: 目标设备节点得最高分
func (p *DeviceUUIDPlugin) Score(...) (int64, *framework.Status) {
    targetUUIDs := p.getPodTargetUUIDs(pod)
    if len(targetUUIDs) == 0 {
        return 50, Success  // 未指定，中性评分
    }
    
    for _, uuid := range targetUUIDs {
        if gpuinfo.GetNodeByDeviceUUID(uuid) == nodeName {
            return 100, Success  // 目标节点最高分
        }
    }
    
    return 0, Success  // 其他节点最低分
}
```

#### 5.2.3 Pod 注解示例

```yaml
# 指定单个设备
metadata:
  annotations:
    scheduler.example.com/target-gpu-uuid: "GPU-abc123-0001"

# 指定多个设备（任选其一）
metadata:
  annotations:
    scheduler.example.com/target-gpu-uuids: "GPU-abc123-0001,GPU-xyz789-0002"
```

---

## 6. 调度流程详解

### 6.1 完整调度流程

```mermaid
flowchart TB
    START([Pod 进入调度队列])
    
    subgraph INIT [初始化阶段]
        INFORMER[Informer 监听到 Pod]
        QUEUE[加入优先队列]
    end
    
    subgraph FILTER [预选阶段 - Filter]
        F0[检查节点可调度性]
        F1[污点容忍检查]
        F2[节点亲和性检查]
        F3[GPU 资源检查]
        F4{GPUUtilization 插件}
        F5{DeviceUUID 插件}
        F6[资源匹配检查]
        
        F4 --> F4A{GPU 健康?}
        F4A -->|是| F4B{显存充足?}
        F4A -->|否| F4X[节点被过滤]
        F4B -->|是| F5
        F4B -->|否| F4X
        
        F5 --> F5A{Pod 指定 UUID?}
        F5A -->|是| F5B{节点包含目标设备?}
        F5A -->|否| F6
        F5B -->|是| F6
        F5B -->|否| F5X[节点被过滤]
    end
    
    subgraph SCORE [优选阶段 - Score]
        S1[DeviceUUID 打分<br/>目标节点: 100<br/>其他: 0]
        S2[GPUUtilization 打分<br/>Score = 100 - 利用率]
        S3[其他内置插件打分]
        S4[加权汇总<br/>最终分数]
        
        S1 --> S4
        S2 --> S4
        S3 --> S4
    end
    
    subgraph BIND [绑定阶段]
        SELECT[选择最高分节点]
        RESERVE[资源预留]
        BIND_ACTION[执行绑定]
        UPDATE[更新 Pod 状态]
    end
    
    START --> INFORMER --> QUEUE
    QUEUE --> F0 --> F1 --> F2 --> F3 --> F4
    F6 --> S1
    S4 --> SELECT --> RESERVE --> BIND_ACTION --> UPDATE
    UPDATE --> END([调度完成])
    
    F4X --> CHECK{还有候选节点?}
    F5X --> CHECK
    CHECK -->|是| S1
    CHECK -->|否| FAIL([调度失败<br/>等待重试])
    
    style START fill:#e8f5e9
    style END fill:#c8e6c9
    style FAIL fill:#ffcdd2
    style F4 fill:#fff3e0
    style F5 fill:#fff3e0
    style S1 fill:#e3f2fd
    style S2 fill:#e3f2fd
```

### 6.2 多插件协作示例

```mermaid
graph TB
    subgraph "Filter 阶段"
        F1[NodeUnschedulable<br/>排除: 不可调度节点]
        F2[TaintToleration<br/>排除: 污点不匹配节点]
        F3[NodeAffinity<br/>排除: 亲和性不匹配节点]
        F4[GPUUtilizationBalancer<br/>排除: GPU 不健康/显存不足]
        F5[DeviceUUIDMatcher<br/>排除: 不含目标设备节点]
    end
    
    subgraph "候选节点池"
        N1[Node-A<br/>GPU: 20%, 包含 GPU-001]
        N2[Node-B<br/>GPU: 80%, 包含 GPU-003]
        N3[Node-C<br/>GPU: 45%, 无目标设备]
    end
    
    subgraph "Score 阶段"
        S1[DeviceUUIDMatcher<br/>权重: 100]
        S2[GPUUtilizationBalancer<br/>权重: 50]
        S3[NodeResourcesBalanced<br/>权重: 1]
    end
    
    subgraph "最终分数"
        R1["Node-A: 100×0 + 50×80 + 1×60<br/>= 4060"]
        R2["Node-B: 100×100 + 50×20 + 1×40<br/>= 10040 ✓"]
        R3["Node-C: 100×0 + 50×55 + 1×50<br/>= 2800"]
    end
    
    F1 --> F2 --> F3 --> F4 --> F5
    F5 --> N1 & N2 & N3
    
    N1 --> S1 & S2 & S3
    N2 --> S1 & S2 & S3
    N3 --> S1 & S2 & S3
    
    S1 --> R1 & R2 & R3
    S2 --> R1 & R2 & R3
    S3 --> R1 & R2 & R3
    
    R2 --> WIN[Node-B 被选中]
    
    style F4 fill:#fff3e0
    style F5 fill:#fff3e0
    style S1 fill:#e3f2fd
    style S2 fill:#e3f2fd
    style R2 fill:#c8e6c9
    style WIN fill:#4caf50
```

---

## 7. 配置与部署

### 7.1 调度器配置文件

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: gpu-scheduler    # Pod 通过此名称指定调度器
    
    plugins:
      # Filter 阶段：按顺序执行
      filter:
        enabled:
          - name: DeviceUUIDMatcher      # 设备 UUID 过滤
          - name: GPUUtilizationBalancer # GPU 利用率过滤
          - name: NodeUnschedulable      # 内置：节点可调度性
          - name: NodeResourcesFit       # 内置：资源匹配
      
      # Score 阶段：并行执行后加权
      score:
        enabled:
          - name: DeviceUUIDMatcher
            weight: 100                   # 最高权重：精确绑定
          - name: GPUUtilizationBalancer
            weight: 50                    # 中等权重：负载均衡
          - name: NodeResourcesBalancedAllocation
            weight: 1                     # 默认权重
```

### 7.2 权重影响分析

```mermaid
pie title Score 权重分布
    "DeviceUUIDMatcher (100)" : 100
    "GPUUtilizationBalancer (50)" : 50
    "NodeResourcesBalanced (1)" : 1
```

| 插件 | 权重 | 影响程度 | 说明 |
|------|------|----------|------|
| DeviceUUIDMatcher | 100 | 决定性 | 指定设备时，目标节点必定被选中 |
| GPUUtilizationBalancer | 50 | 显著 | 未指定设备时，主导负载均衡 |
| NodeResourcesBalanced | 1 | 辅助 | 默认资源均衡策略 |

### 7.3 部署架构

```mermaid
graph TB
    subgraph "Kubernetes 集群"
        subgraph "kube-system namespace"
            SA[ServiceAccount<br/>gpu-scheduler]
            RBAC[ClusterRole/Binding]
            CM[ConfigMap<br/>scheduler-config]
            
            subgraph "调度器 Pod (HA)"
                S1[gpu-scheduler-1<br/>Leader]
                S2[gpu-scheduler-2<br/>Standby]
            end
        end
        
        subgraph "工作节点"
            N1[Node-A<br/>GPU: 2x A100]
            N2[Node-B<br/>GPU: 2x A100]
            N3[Node-C<br/>GPU: 2x A100]
        end
        
        API[API Server]
        PROM[Prometheus<br/>GPU 指标]
    end
    
    SA --> RBAC
    RBAC --> API
    CM --> S1 & S2
    S1 & S2 --> API
    API --> N1 & N2 & N3
    PROM -.-> S1 & S2
    S1 -.-> N1 & N2 & N3
    
    style S1 fill:#c8e6c9
    style PROM fill:#fff3e0
```

---

## 8. 使用示例

### 8.1 场景一：GPU 负载均衡

```yaml
# 自动调度到 GPU 利用率最低的节点
apiVersion: v1
kind: Pod
metadata:
  name: training-job
spec:
  schedulerName: gpu-scheduler    # 使用 GPU 调度器
  containers:
    - name: training
      image: pytorch/pytorch:latest
      command: ["python", "train.py"]
      resources:
        limits:
          nvidia.com/gpu: 1
          memory: "16Gi"
```

**调度结果：**

```mermaid
graph LR
    subgraph "调度前"
        A1[Node-A<br/>GPU: 10%]
        A2[Node-B<br/>GPU: 90%]
        A3[Node-C<br/>GPU: 60%]
    end
    
    subgraph "调度后"
        B1[Node-A<br/>GPU: 25%<br/>✅ 新任务]
        B2[Node-B<br/>GPU: 90%]
        B3[Node-C<br/>GPU: 60%]
    end
    
    A1 --> B1
    A2 --> B2
    A3 --> B3
    
    style B1 fill:#c8e6c9
```

### 8.2 场景二：指定 GPU 设备

```yaml
# 绑定到特定 GPU 设备
apiVersion: v1
kind: Pod
metadata:
  name: debug-job
  annotations:
    scheduler.example.com/target-gpu-uuid: "GPU-abc123-0001"
spec:
  schedulerName: gpu-scheduler
  containers:
    - name: debug
      image: nvidia/cuda:12.0
      command: ["cuda-gdb", "./program"]
      resources:
        limits:
          nvidia.com/gpu: 1
```

**调度流程：**

```mermaid
sequenceDiagram
    participant P as Pod
    participant S as GPU Scheduler
    participant NA as Node-A<br/>GPU-abc123-0002
    participant NB as Node-B<br/>GPU-abc123-0001
    
    P->>S: 调度请求<br/>target-gpu-uuid: GPU-abc123-0001
    
    S->>NA: Filter: 包含 GPU-abc123-0001?
    NA-->>S: ❌ 不包含
    
    S->>NB: Filter: 包含 GPU-abc123-0001?
    NB-->>S: ✅ 包含
    
    S->>NA: Score: 0
    S->>NB: Score: 100
    
    S->>NB: Bind Pod → Node-B
    
    Note over P,NB: 调度成功
```

### 8.3 场景三：混合策略

```yaml
# 指定多个设备，选择其中负载最低的
apiVersion: v1
kind: Pod
metadata:
  name: inference-job
  annotations:
    # 可接受的设备列表
    scheduler.example.com/target-gpu-uuids: "GPU-001,GPU-002,GPU-003"
spec:
  schedulerName: gpu-scheduler
  containers:
    - name: inference
      image: tensorflow/serving:latest
      resources:
        limits:
          nvidia.com/gpu: 1
```

**调度逻辑：**

1. **Filter**: 保留包含 GPU-001 或 GPU-002 或 GPU-003 的节点
2. **Score**: 
   - DeviceUUIDMatcher: 所有候选节点都得 100 分
   - GPUUtilizationBalancer: 在候选节点中选择利用率最低的

```mermaid
graph TB
    subgraph "Filter 阶段"
        N1[Node-A<br/>GPU-001: 80%]
        N2[Node-B<br/>GPU-002: 20%]
        N3[Node-C<br/>GPU-003: 50%]
        N4[Node-D<br/>GPU-004: 10%]
    end
    
    subgraph "候选节点"
        C1[Node-A ✓]
        C2[Node-B ✓]
        C3[Node-C ✓]
    end
    
    subgraph "Score 阶段"
        S1["Node-A: 100×100 + 50×(100-80) = 11000"]
        S2["Node-B: 100×100 + 50×(100-20) = 15000 ✓"]
        S3["Node-C: 100×100 + 50×(100-50) = 12500"]
    end
    
    N1 --> C1
    N2 --> C2
    N3 --> C3
    N4 -.->|被过滤| X[❌]
    
    C1 --> S1
    C2 --> S2
    C3 --> S3
    
    S2 --> WIN[Node-B 被选中]
    
    style N4 fill:#ffcdd2
    style S2 fill:#c8e6c9
    style WIN fill:#4caf50
```

---

## 附录

### A. 关键接口速查

```go
// ============================================
// 插件必须实现的接口
// ============================================

// 基础接口
type Plugin interface {
    Name() string
}

// Filter 插件（可选实现）
type FilterPlugin interface {
    Plugin
    Filter(ctx, state, pod, nodeInfo) *Status
}

// Score 插件（可选实现）
type ScorePlugin interface {
    Plugin
    Score(ctx, state, pod, nodeName) (int64, *Status)
    ScoreExtensions() ScoreExtensions  // 可返回 nil
}

// Reserve 插件（可选实现）
type ReservePlugin interface {
    Plugin
    Reserve(ctx, state, pod, nodeName) *Status
    Unreserve(ctx, state, pod, nodeName)
}

// PreBind 插件（可选实现）
type PreBindPlugin interface {
    Plugin
    PreBind(ctx, state, pod, nodeName) *Status
}

// Bind 插件（可选实现，通常使用默认）
type BindPlugin interface {
    Plugin
    Bind(ctx, state, pod, nodeName) *Status
}
```

### B. 状态码说明

| 状态 | 值 | 含义 | Filter 行为 | Score 行为 |
|------|-----|------|-------------|------------|
| Success | 0 | 成功 | 节点通过 | 分数有效 |
| Error | 1 | 内部错误 | 调度终止 | 调度终止 |
| Unschedulable | 2 | 不可调度 | 节点被过滤 | - |
| UnschedulableAndUnresolvable | 3 | 永久不可调度 | 节点被过滤，不重试 | - |
| Wait | 4 | 等待 | - | 等待条件满足 |
| Skip | 5 | 跳过 | - | 使用默认分数 |

### C. 参考资料

- [Kubernetes 调度框架官方文档](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/)
- [调度器插件开发指南](https://kubernetes.io/docs/reference/scheduling/config/)
- [NVIDIA GPU Device Plugin](https://github.com/NVIDIA/k8s-device-plugin)
- [DCGM Exporter](https://github.com/NVIDIA/dcgm-exporter)

---

> 本文档为学习 Kubernetes 调度器扩展机制而编写，展示了如何通过调度框架实现 GPU 感知调度。