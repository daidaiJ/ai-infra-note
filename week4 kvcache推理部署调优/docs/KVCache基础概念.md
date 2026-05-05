# KVCache 基础概念详解

> 理解 KVCache 存储原理、缓存复用机制以及在 LLM 推理中的关键作用。

---

## 目录

- [1. KVCache 概述](#1-kvcache-概述)
- [2. KVCache 存储架构](#2-kvcache-存储架构)
- [3. 缓存复用原理](#3-缓存复用原理)
- [4. PD 分离架构](#4-pd-分离架构)
- [5. 性能影响分析](#5-性能影响分析)
- [附录](#附录)

---

## 1. KVCache 概述

### 1.1 什么是 KVCache

KVCache（Key-Value Cache）是 Transformer 模型推理过程中的核心优化技术。在自回归生成过程中，模型需要反复访问之前计算过的 Key 和 Value 状态。

```mermaid
graph TB
    subgraph "Transformer 推理过程"
        T1["输入Token"]
        T2["Self-Attention<br/>计算 Q/K/V"]
        T3["生成下一个Token"]
        T4["KVCache存储<br/>K/V状态"]
    end

    T1 --> T2 --> T3
    T2 --> T4
    T4 -->|"复用"| T2

    style T4 fill:#c8e6c9
```

**核心作用：**

| 作用 | 说明 |
|------|------|
| **避免重复计算** | 已计算的 K/V 状态缓存后可直接复用 |
| **降低计算量** | 每次推理只需计算新 Token 的 Attention |
| **加速生成** | 显著减少 Prefill 后的 Decode 时间 |

### 1.2 KVCache 的内存消耗

```mermaid
graph LR
    subgraph "KVCache内存占用计算"
        M1["模型参数"]
        M2["序列长度"]
        M3["隐藏层维度"]
        M4["层数"]
    end

    M1 --> C["KVCache大小<br/>= 2 × L × S × D × H"]
    M2 --> C
    M3 --> C
    M4 --> C

    style C fill:#fff3e0
```

**计算公式：**

```
KVCache大小 = 2 × 层数(L) × 序列长度(S) × 隐藏维度(D) × 头数(H) × 精度(bytes)

示例：Qwen2.5-7B，序列长度4096，FP16
- 层数 L = 28
- 隐藏维度 D = 3584
- 头数 H = 28
- 精度 = 2 bytes (FP16)

KVCache ≈ 2 × 28 × 4096 × 3584 × 28 × 2 ≈ 16.4 GB
```

**内存压力分析：**

| 序列长度 | KVCache大小 (Qwen2.5-7B FP16) | GPU显存占比 (24GB) |
|----------|-------------------------------|---------------------|
| 1024 | ~4.1 GB | 17% |
| 4096 | ~16.4 GB | 68% |
| 8192 | ~32.8 GB | **超出显存** |
| 32768 | ~131 GB | **严重超出** |

> **结论：** 长上下文场景下，KVCache 成为 GPU 显存的主要瓶颈。

---

## 2. KVCache 存储架构

### 2.1 多级存储层次

```mermaid
graph TB
    subgraph "三级存储架构"
        L1["L1: GPU显存<br/>最快<br/>容量最小"]
        L2["L2: 主机内存<br/>较快<br/>容量中等"]
        L3["L3: 分布式存储<br/>较慢<br/>容量最大"]
    end

    L1 -->|"写回卸载"| L2
    L2 -->|"写回卸载"| L3
    L3 -->|"预取加载"| L2
    L2 -->|"加载"| L1

    style L1 fill:#e3f2fd
    style L2 fill:#fff3e0
    style L3 fill:#c8e6c9
```

**各级存储特性：**

| 存储层级 | 容量 | 带宽 | 延迟 | 成本 |
|----------|------|------|------|------|
| **L1 GPU显存** | 24-80GB | ~1TB/s | ~1μs | 最高 |
| **L2 主机内存** | 128-512GB | ~100GB/s | ~10μs | 中等 |
| **L3 分布式存储** | TB-PB级 | ~10GB/s | ~100μs | 较低 |

### 2.2 为什么需要 KVCache 卸载

```mermaid
graph TB
    subgraph "问题：GPU显存瓶颈"
        P1["长上下文请求"]
        P2["多并发请求"]
        P3["KVCache累积"]
        P4["显存OOM"]
    end

    P1 --> P3
    P2 --> P3
    P3 --> P4

    style P4 fill:#ffcdd2
```

**核心痛点：**

| 痛点 | 说明 | 影响 |
|------|------|------|
| **显存容量限制** | GPU显存无法容纳长序列KVCache | 长上下文请求失败 |
| **并发受限** | 多请求KVCache共享显存空间 | 吞吐量下降 |
| **成本高昂** | 大显存GPU价格昂贵 | 部署成本增加 |

**解决方案：KVCache卸载**

```mermaid
graph LR
    subgraph "解决方案"
        S1["卸载到主机内存"]
        S2["卸载到分布式存储"]
        S3["跨实例共享"]
    end

    subgraph "效果"
        E1["突破显存瓶颈"]
        E2["支持长上下文"]
        E3["提升并发能力"]
    end

    S1 --> E1
    S2 --> E2
    S3 --> E3

    style S1 fill:#c8e6c9
    style S2 fill:#c8e6c9
    style S3 fill:#c8e6c9
```

---

## 3. 缓存复用原理

### 3.1 Prefix Caching 机制

当多个请求共享相同的前缀（如 System Prompt、聊天历史）时，KVCache 完全相同，可以复用。

```mermaid
graph TB
    subgraph "请求场景"
        R1["请求A<br/>System Prompt + 用户问题A"]
        R2["请求B<br/>System Prompt + 用户问题B"]
        R3["请求C<br/>System Prompt + 用户问题C"]
    end

    subgraph "KVCache复用"
        K1["共享前缀KVCache<br/>只计算一次"]
        K2["各自后缀KVCache<br/>分别计算"]
    end

    R1 -->|"共享"| K1
    R2 -->|"共享"| K1
    R3 -->|"共享"| K1

    K1 --> K2

    style K1 fill:#c8e6c9
```

**复用收益：**

| 场景 | 无复用计算量 | 有复用计算量 | 收益 |
|------|-------------|-------------|------|
| 100个请求共享1000 tokens前缀 | 100×1000 = 100K tokens | 1000 + 100×100 = 11K tokens | **91%减少** |
| 多轮对话（10轮，每轮+500 tokens） | 500+1000+...+5000 = 27500 | 累计复用 | **显著减少** |

### 3.2 RadixAttention 原理

```mermaid
graph TB
    subgraph "RadixTree结构"
        N1["根节点<br/>空前缀"]
        N2["节点A<br/>'你是AI助手'"]
        N3["节点B<br/>'请回答问题'"]
        N4["叶子A1<br/>'问题1'"]
        N5["叶子A2<br/>'问题2'"]
    end

    N1 --> N2 --> N3
    N3 --> N4
    N3 --> N5

    style N2 fill:#e3f2fd
    style N3 fill:#fff3e0
    style N4 fill:#c8e6c9
    style N5 fill:#c8e6c9
```

**RadixAttention 特点：**

| 特点 | 说明 |
|------|------|
| **前缀树组织** | KVCache按前缀树结构组织 |
| **自动匹配** | 新请求自动匹配最长公共前缀 |
| **增量计算** | 只需计算前缀后的新Token |
| **内存共享** | 相同前缀共享同一份KVCache |

---

## 4. PD 分离架构

### 4.1 Prefill 与 Decode 的差异

```mermaid
graph TB
    subgraph "Prefill阶段"
        P1["处理整个输入序列"]
        P2["计算密集型"]
        P3["KVCache全部生成"]
        P4["时间较长"]
    end

    subgraph "Decode阶段"
        D1["逐Token生成"]
        D2["内存密集型"]
        D3["KVCache增量更新"]
        D4["时间分散"]
    end

    P1 --> P2 --> P3 --> P4
    D1 --> D2 --> D3 --> D4

    style P1 fill:#e3f2fd
    style D1 fill:#fff3e0
```

**阶段特性对比：**

| 特性 | Prefill | Decode |
|------|---------|--------|
| **计算模式** | 批量并行计算 | 逐Token串行计算 |
| **资源需求** | GPU计算能力 | GPU显存容量 |
| **时间分布** | 集中在前 | 分散在整个生成过程 |
| **KVCache** | 全量生成 | 增量更新 |

### 4.2 PD分离架构原理

```mermaid
graph TB
    subgraph "传统架构"
        T1["单实例<br/>Prefill + Decode"]
        T2["资源竞争<br/>计算与显存冲突"]
    end

    subgraph "PD分离架构"
        P["Prefill节点<br/>GPU计算密集"]
        D["Decode节点<br/>显存容量密集"]
        T["KVCache传输<br/>RDMA零拷贝"]
    end

    T1 --> T2

    P -->|"KVCache"| T --> D

    style T2 fill:#ffcdd2
    style P fill:#e3f2fd
    style D fill:#fff3e0
    style T fill:#c8e6c9
```

**PD分离优势：**

| 优势 | 说明 |
|------|------|
| **资源独立** | Prefill节点专注计算，Decode节点专注显存 |
| **性能稳定** | Decode阶段不受Prefill计算干扰 |
| **弹性扩展** | 可独立扩展Prefill或Decode节点数量 |
| **KVCache共享** | 多Decode节点可共享同一Prefill结果 |

### 4.3 KVCache传输机制

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Prefill as Prefill节点
    participant Storage as KVCache存储
    participant Decode as Decode节点

    Note over Client,Decode: === PD分离流程 ===

    Client->>Prefill: 发送请求
    Prefill->>Prefill: 执行Prefill计算
    Prefill->>Storage: 存储KVCache（RDMA）
    Prefill-->>Client: 返回Prefill完成

    Storage->>Decode: 传输KVCache（预取）
    Decode->>Decode: 加载KVCache到显存
    Decode->>Decode: 执行Decode生成
    Decode-->>Client: 返回生成结果

    Note over Storage,Decode: RDMA零拷贝传输
```

**传输关键技术：**

| 技术 | 说明 | 效果 |
|------|------|------|
| **RDMA零拷贝** | 直接内存到内存传输 | 延迟~2μs，带宽~100Gbps |
| **GPUDirect** | GPU显存直接参与RDMA | 避免CPU中转开销 |
| **NUMA亲和** | 网卡与KVCache进程同NUMA | 避免跨NUMA延迟 |

---

## 5. 性能影响分析

### 5.1 KVCache对推理性能的影响

```mermaid
graph TB
    subgraph "无KVCache优化"
        N1["每次请求完整Prefill"]
        N2["重复计算相同前缀"]
        N3["延迟高"]
        N4["吞吐低"]
    end

    subgraph "有KVCache优化"
        Y1["复用共享前缀KVCache"]
        Y2["只计算新Token"]
        Y3["延迟低"]
        Y4["吞吐高"]
    end

    N1 --> N2 --> N3 --> N4
    Y1 --> Y2 --> Y3 --> Y4

    style N3 fill:#ffcdd2
    style N4 fill:#ffcdd2
    style Y3 fill:#c8e6c9
    style Y4 fill:#c8e6c9
```

**性能提升案例：**

| 场景 | 无KVCache复用 | 有KVCache复用 | 提升 |
|------|--------------|--------------|------|
| 1000 tokens共享前缀 + 100 tokens问题 | Prefill 1100 tokens | Prefill 100 tokens | **91%延迟降低** |
| 10轮多轮对话 | 每轮完整Prefill | 累积复用历史KV | **延迟递减** |
| 100并发共享前缀 | 100×Prefill | 1×Prefill + 100×Decode | **吞吐提升10x** |

### 5.2 存储层级对性能的影响

```mermaid
graph TB
    subgraph "存储位置选择"
        S1["全部GPU显存"]
        S2["卸载到主机内存"]
        S3["卸载到分布式存储"]
    end

    subgraph "性能影响"
        P1["容量小<br/>速度最快"]
        P2["容量中等<br/>速度中等"]
        P3["容量大<br/>速度较慢"]
    end

    S1 --> P1
    S2 --> P2
    S3 --> P3

    style S1 fill:#e3f2fd
    style S2 fill:#fff3e0
    style S3 fill:#c8e6c9
```

**选择策略：**

| 场景 | 推荐存储位置 | 原因 |
|------|-------------|------|
| 短序列单请求 | GPU显存 | 延迟最低 |
| 长序列单请求 | 主机内存 | 突破显存限制 |
| 多实例共享前缀 | 分布式存储 | 跨实例复用 |
| PD分离架构 | RDMA传输 | 高效跨节点传输 |

---

## 附录

### A. KVCache相关术语

| 术语 | 说明 |
|------|------|
| **KVCache** | Key-Value Cache，Transformer中的注意力状态缓存 |
| **Prefill** | 预填充阶段，处理输入序列并生成初始KVCache |
| **Decode** | 解码阶段，逐Token生成并更新KVCache |
| **Prefix Caching** | 前缀缓存，共享前缀的KVCache复用 |
| **RadixAttention** | 基于前缀树组织的KVCache匹配机制 |
| **PD分离** | Prefill与Decode阶段分离到不同节点 |
| **KV卸载** | 将KVCache从GPU卸载到其他存储 |
| **RDMA** | Remote Direct Memory Access，远程直接内存访问 |

### B. 性能指标速查

| 指标 | GPU显存 | 主机内存 | 分布式存储 |
|------|---------|----------|-----------|
| **带宽** | ~1TB/s | ~100GB/s | ~10GB/s |
| **延迟** | ~1μs | ~10μs | ~100μs |
| **容量** | 24-80GB | 128-512GB | TB-PB |

### C. 参考资料

- [Mooncake GitHub](https://github.com/kvcache-ai/Mooncake)
- [LMCache Documentation](https://docs.lmcache.ai/)
- [HiCache Design](https://docs.sglang.io/docs/advanced_features/hicache_design)
- [vLLM KVCache Documentation](https://docs.vllm.ai/)

---

> 下一节：[推理引擎集成对比.md](推理引擎集成对比.md) - 了解 vLLM/SGLang 与 KVCache方案的集成方式。