# RDMA 基础概念详解

> 理解 RDMA 如何通过零拷贝和内核旁路实现低延迟，以及在 AI 推理/训练场景中的应用价值。

---

## 目录

- [1. 什么是 RDMA](#1-什么是-rdma)
- [2. RDMA 的核心优势](#2-rdma-的核心优势)
- [3. RoCE vs InfiniBand](#3-roce-vs-infiniband)
- [4. RDMA 在 AI 场景的应用](#4-rdma-在-ai-场景的应用)
- [5. Kubernetes 中的 RDMA](#5-kubernetes-中的-rdma)

---

## 1. 什么是 RDMA

### 1.1 定义

**RDMA (Remote Direct Memory Access)**：远程直接内存访问，是一种让网络设备直接读写远程主机内存的技术，无需操作系统内核参与。

### 1.2 传统网络 vs RDMA

```mermaid
graph TB
    subgraph "传统网络传输"
        T1["应用层<br/>用户缓冲区"]
        T2["内核层<br/>Socket缓冲区"]
        T3["内核层<br/>TCP/IP协议栈"]
        T4["驱动层<br/>网卡驱动"]
        T5["硬件层<br/>网卡"]
        T6["网络传输"]
        T7["对方网卡"]
        T8["对方内核"]
        T9["对方应用"]
        
        T1 -->|"数据拷贝1"| T2
        T2 -->|"协议处理"| T3
        T3 -->|"数据拷贝2"| T4
        T4 --> T5
        T5 --> T6 --> T7
        T7 --> T8 --> T9
    end

    subgraph "RDMA传输"
        R1["应用层<br/>用户缓冲区"]
        R2["硬件层<br/>RDMA网卡"]
        R3["网络传输"]
        R4["对方网卡"]
        R5["对方内存"]
        R6["对方应用"]
        
        R1 -->|"直接DMA"| R2
        R2 --> R3 --> R4
        R4 -->|"直接写入"| R5
        R5 --> R6
    end

    style T1 fill:#ffcdd2
    style T2 fill:#ffcdd2
    style T3 fill:#ffcdd2
    style T4 fill:#ffcdd2
    
    style R1 fill:#c8e6c9
    style R2 fill:#c8e6c9
```

### 1.3 关键差异对比

| 特性 | 传统网络 | RDMA |
|------|----------|------|
| **数据拷贝** | 至少 2 次（用户→内核→网卡） | 0 次（直接 DMA） |
| **内核参与** | 每次传输都需要 | 完全旁路 |
| **上下文切换** | 需要切换到内核态 | 无切换 |
| **CPU 开销** | 高（处理协议栈） | 低（网卡硬件处理） |
| **延迟** | 50-100μs | 1-2μs |
| **带宽利用率** | 60-70% | 95%+ |

---

## 2. RDMA 的核心优势

### 2.1 零拷贝 (Zero Copy)

```mermaid
graph LR
    subgraph "传统方式：多次拷贝"
        A1["用户缓冲区"] -->|"拷贝"| A2["内核缓冲区"]
        A2 -->|"拷贝"| A3["网卡缓冲区"]
        A3 --> A4["网络发送"]
    end

    subgraph "RDMA：零拷贝"
        B1["用户缓冲区"] -->|"DMA直接读取"| B2["网卡"]
        B2 --> B3["网络发送"]
    end

    style A1 fill:#ffcdd2
    style A2 fill:#ffcdd2
    style A3 fill:#ffcdd2
    
    style B1 fill:#c8e6c9
    style B2 fill:#c8e6c9
```

**核心原理：**
- 应用程序直接在内存中准备数据
- RDMA 网卡通过 DMA 直接读取该内存区域
- 无需经过内核缓冲区

### 2.2 内核旁路 (Kernel Bypass)

```mermaid
graph TB
    subgraph "传统网络路径"
        K1["应用程序"]
        K2["系统调用<br/>send()/recv()"]
        K3["内核态"]
        K4["Socket缓冲区"]
        K5["TCP/IP协议栈"]
        K6["网卡驱动"]
        K7["网卡"]
        
        K1 --> K2 --> K3
        K3 --> K4 --> K5 --> K6 --> K7
    end

    subgraph "RDMA路径"
        R1["应用程序"]
        R2["RDMA库<br/>ibv_post_send()"]
        R3["用户态"]
        R4["RDMA网卡"]
        
        R1 --> R2 --> R3
        R3 -->|"直接命令"| R4
    end

    style K2 fill:#ffcdd2
    style K3 fill:#ffcdd2
    style K4 fill:#ffcdd2
    style K5 fill:#ffcdd2
    style K6 fill:#ffcdd2

    style R2 fill:#c8e6c9
    style R3 fill:#c8e6c9
    style R4 fill:#c8e6c9
```

**核心原理：**
- 应用程序通过用户态库（如 libibverbs）直接操作网卡
- 无需系统调用进入内核
- 无需内核协议栈处理

### 2.3 延迟优势

| 场景 | 传统 TCP | RDMA | 优势倍数 |
|------|----------|------|----------|
| **单节点内存访问** | ~100μs | ~1μs | 100x |
| **跨节点小消息** | ~50μs | ~2μs | 25x |
| **大块数据传输** | 带宽利用率 60% | 带宽利用率 95%+ | 1.5x+ |

---

## 3. RoCE vs InfiniBand

### 3.1 两种 RDMA 网络

```mermaid
graph TB
    subgraph "InfiniBand网络"
        IB1["IB网卡"]
        IB2["IB交换机"]
        IB3["IB子网管理器<br/>OpenSM"]
        IB4["专用IB链路"]
        
        IB1 --> IB4 --> IB2 --> IB4 --> IB1
        IB3 -->|"管理"| IB2
    end

    subgraph "RoCE网络"
        R1["RoCE网卡"]
        R2["以太网交换机"]
        R3["IP网络"]
        R4["PFC/ECN流控"]
        
        R1 --> R3 --> R2 --> R3 --> R1
        R4 -->|"无损网络"| R2
    end

    style IB1 fill:#e3f2fd
    style IB2 fill:#e3f2fd
    style IB3 fill:#e3f2fd
    
    style R1 fill:#c8e6c9
    style R2 fill:#c8e6c9
    style R4 fill:#fff3e0
```

### 3.2 核心差异对比

| 特性 | InfiniBand | RoCE |
|------|------------|------|
| **网络类型** | 专用网络 | 以太网（需无损配置） |
| **协议** | IB协议（原生RDMA） | UDP封装RDMA |
| **流控** | 链路层流控（内置） | 需配置PFC/ECN |
| **成本** | 高（专用设备） | 低（以太网设备） |
| **带宽** | 100G-400G | 25G-100G（常见） |
| **延迟** | ~1μs | ~2μs |
| **部署复杂度** | 中（需SM） | 高（需无损以太网） |

### 3.3 适用场景

```mermaid
graph TB
    subgraph "选择InfiniBand"
        S1["大规模训练集群<br/>1000+节点"]
        S2["极致延迟要求<br/><1μs"]
        S3["已有IB基础设施"]
    end

    subgraph "选择RoCE"
        S4["中小规模集群<br/>10-100节点"]
        S5["成本敏感"]
        S6["已有以太网基础设施"]
    end

    style S1 fill:#e3f2fd
    style S2 fill:#e3f2fd
    style S3 fill:#e3f2fd
    
    style S4 fill:#c8e6c9
    style S5 fill:#c8e6c9
    style S6 fill:#c8e6c9
```

### 3.4 RoCE 的无损网络要求

**为什么 RoCE 需要无损网络？**

```mermaid
graph TB
    subgraph "问题：丢包导致性能下降"
        P1["以太网丢包"]
        P2["RDMA重传"]
        P3["Go-Back-N策略<br/>重传大量数据"]
        P4["延迟暴涨<br/>从2μs→ms级"]
        
        P1 --> P2 --> P3 --> P4
    end

    subgraph "解决：无损以太网"
        S1["PFC<br/>Priority Flow Control"]
        S2["ECN<br/>Explicit Congestion Notification"]
        S3["DCB<br/>Data Center Bridging"]
        S4["零丢包<br/>RDMA性能稳定"]
        
        S1 --> S4
        S2 --> S4
        S3 --> S4
    end

    style P1 fill:#ffcdd2
    style P2 fill:#ffcdd2
    style P3 fill:#ffcdd2
    style P4 fill:#ffcdd2

    style S1 fill:#c8e6c9
    style S2 fill:#c8e6c9
    style S4 fill:#c8e6c9
```

---

## 4. RDMA 在 AI 场景的应用

### 4.1 多节点推理服务

```mermaid
graph TB
    subgraph "场景：分布式推理"
        N1["推理节点1<br/>模型分片A"]
        N2["推理节点2<br/>模型分片B"]
        N3["推理节点3<br/>模型分片C"]
        
        N1 -->|"中间结果<br/>RDMA传输"| N2
        N2 -->|"中间结果<br/>RDMA传输"| N3
        N3 -->|"最终结果"| Output["输出"]
    end

    subgraph "性能影响"
        P1["传统TCP<br/>50μs延迟"]
        P2["RDMA<br/>2μs延迟"]
        P3["推理延迟降低<br/>25x"]
        
        P1 -.->|"对比"| P2 --> P3
    end

    style N1 fill:#e3f2fd
    style N2 fill:#e3f2fd
    style N3 fill:#e3f2fd
    
    style P2 fill:#c8e6c9
    style P3 fill:#c8e6c9
```

### 4.2 分布式训练

```mermaid
graph TB
    subgraph "训练过程中的通信"
        G1["GPU0<br/>梯度计算"]
        G2["GPU1<br/>梯度计算"]
        G3["GPU2<br/>梯度计算"]
        G4["GPU3<br/>梯度计算"]
        
        G1 -->|"梯度传输"| NCCL["NCCL AllReduce"]
        G2 -->|"梯度传输"| NCCL
        G3 -->|"梯度传输"| NCCL
        G4 -->|"梯度传输"| NCCL
        
        NCCL -->|"RDMA<br/>低延迟"| Aggregate["梯度聚合"]
    end

    subgraph "RDMA优势"
        A1["梯度同步延迟降低"]
        A2["训练吞吐提升"]
        A3["GPU利用率提高"]
        
        Aggregate --> A1 --> A2 --> A3
    end

    style G1 fill:#fff3e0
    style G2 fill:#fff3e0
    style G3 fill:#fff3e0
    style G4 fill:#fff3e0
    
    style NCCL fill:#e3f2fd
    style Aggregate fill:#c8e6c9
```

### 4.3 NCCL 与 RDMA

**NCCL (NVIDIA Collective Communication Library)** 是 NVIDIA 提供的集合通信库，专门优化 GPU 集群间的通信。

```mermaid
graph TB
    subgraph "NCCL通信路径"
        N1["GPU显存"]
        N2["GPU Direct RDMA<br/>直接传输"]
        N3["RDMA网卡"]
        N4["网络传输"]
        N5["对方网卡"]
        N6["对方GPU显存"]
        
        N1 -->|"GPU Direct"| N2 --> N3
        N3 --> N4 --> N5
        N5 -->|"直接写入"| N6
    end

    subgraph "关键技术"
        T1["GPUDirect RDMA<br/>网卡直接访问GPU显存"]
        T2["GPUDirect NVLink<br/>GPU间直连"]
        T3["NCCL自动选择<br/>最优传输路径"]
        
        N2 -.-> T1
    end

    style N1 fill:#fff3e0
    style N2 fill:#e3f2fd
    style N3 fill:#c8e6c9
    style N6 fill:#fff3e0
```

---

## 5. Kubernetes 中的 RDMA

### 5.1 核心挑战

```mermaid
graph TB
    subgraph "挑战"
        C1["设备发现<br/>如何发现节点上的RDMA网卡"]
        C2["资源管理<br/>如何注册RDMA资源"]
        C3["设备分配<br/>如何将网卡分配给Pod"]
        C4["配置注入<br/>如何让Pod使用RDMA"]
    end

    style C1 fill:#ffcdd2
    style C2 fill:#ffcdd2
    style C3 fill:#ffcdd2
    style C4 fill:#ffcdd2
```

### 5.2 解决方案概览

```mermaid
graph TB
    subgraph "方案演进"
        S1["Device Plugin<br/>设备发现与注册"]
        S2["NetworkAttachmentDefinition<br/>网络附加定义"]
        S3["CNI插件<br/>设备注入"]
        S4["DRA<br/>声明式资源管理"]
        
        S1 --> S2 --> S3 --> S4
    end

    subgraph "协作关系"
        R1["Device Plugin<br/>发现RDMA设备"]
        R2["NAD<br/>定义网络配置"]
        R3["host-device CNI<br/>注入设备"]
        R4["Pod<br/>使用RDMA"]
        
        R1 -->|"设备信息"| R2
        R2 -->|"配置"| R3
        R3 -->|"设备注入"| R4
    end

    style S1 fill:#c8e6c9
    style S2 fill:#fff3e0
    style S3 fill:#e3f2fd
    style S4 fill:#9c27b0

    style R1 fill:#c8e6c9
    style R2 fill:#fff3e0
    style R3 fill:#e3f2fd
    style R4 fill:#4caf50
```

### 5.3 与 GPU 资源管理的类比

| 概念 | GPU 资源管理 | RDMA 资源管理 |
|------|-------------|---------------|
| **设备发现** | NVIDIA Device Plugin | RDMA Device Plugin |
| **资源注册** | `nvidia.com/gpu` | `rdma/ib` 或 `rdma/roce` |
| **设备分配** | Allocate 接口 | Allocate 接口 |
| **配置注入** | 设备文件 + Env | 设备文件 + NAD |
| **调度扩展** | GPU 拓扑感知 | 网络拓扑感知 |

---

## 附录

### A. RDMA 核心术语

| 术语 | 英文 | 说明 |
|------|------|------|
| **QP** | Queue Pair | RDMA通信的端点，包含发送队列和接收队列 |
| **CQ** | Completion Queue | 完成队列，用于通知操作完成 |
| **MR** | Memory Region | 注册的内存区域，允许网卡访问 |
| **PD** | Protection Domain | 保护域，隔离不同进程的RDMA资源 |
| **WR** | Work Request | 工作请求，描述要执行的RDMA操作 |
| **LID** | Local Identifier | IB网络中的节点标识符 |

### B. 常见 RDMA 设备

| 设备类型 | 设备名示例 | 说明 |
|----------|-----------|------|
| **Mellanox IB** | `mlx5_0`, `mlx5_1` | ConnectX系列IB网卡 |
| **Mellanox RoCE** | `mlx5_0` | ConnectX系列RoCE网卡 |
| **Intel Omni-Path** | `hfi1_0` | Intel OPA网卡 |

### C. 参考资源

- [RDMA-aware Networks Programming User Manual](https://docs.nvidia.com/networking/)
- [NVIDIA NCCL Documentation](https://docs.nvidia.com/deeplearning/nccl/)
- [Mellanox OFED Documentation](https://docs.nvidia.com/networking/software/)
- [RDMA CNI GitHub](https://github.com/Mellanox/rdma-cni)

---

> 下一章：[网络设备制备与映射.md](网络设备制备与映射.md) - 学习 NAD 与 host-device 的完整协作链路。