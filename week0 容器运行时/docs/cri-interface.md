# Kubernetes CRI 接口详解

> 本文档详细介绍 Kubernetes Container Runtime Interface（CRI）的接口定义、调用流程和核心能力。

---

## 目录

- [1. CRI 概述](#1-cri-概述)
- [2. 接口定义](#2-接口定义)
- [3. RuntimeService 接口](#3-runtimeservice-接口)
- [4. ImageService 接口](#4-imageservice-接口)
- [5. Pod 生命周期管理](#5-pod-生命周期管理)
- [6. 容器生命周期管理](#6-容器生命周期管理)
- [7. crictl 工具详解](#7-crictl-工具详解)
- [8. 最佳实践](#8-最佳实践)

---

## 1. CRI 概述

### 1.1 什么是 CRI

CRI（Container Runtime Interface）是 Kubernetes 定义的一组 gRPC 接口，用于 kubelet 与容器运行时之间的通信。

```mermaid
graph TB
    subgraph "Kubernetes 节点"
        KL[kubelet]
        
        subgraph "CRI 层"
            CRI[CRI gRPC Server]
        end
        
        subgraph "运行时实现"
            CT[containerd]
            CO[CRI-O]
            CD[cri-dockerd]
        end
        
        subgraph "底层运行时"
            OCI[OCI Runtime<br/>runc/crun]
        end
    end
    
    KL -->|"gRPC 调用"| CRI
    CRI --> CT
    CRI --> CO
    CRI --> CD
    CT --> OCI
    CO --> OCI
    
    style KL fill:#e3f2fd
    style CRI fill:#fff3e0
    style CT fill:#c8e6c9
    style CO fill:#c8e6c9
    style CD fill:#c8e6c9
    style OCI fill:#fce4ec
```

### 1.2 CRI 设计目标

```mermaid
graph LR
    subgraph "设计目标"
        G1[标准化<br/>统一接口规范]
        G2[解耦<br/>kubelet 与运行时独立]
        G3[可扩展<br/>支持多种运行时]
        G4[简化<br/>减少 kubelet 复杂度]
    end
    
    style G1 fill:#e3f2fd
    style G2 fill:#fff3e0
    style G3 fill:#c8e6c9
    style G4 fill:#fce4ec
```

| 目标 | 说明 | 实现 |
|------|------|------|
| **标准化** | 定义统一的运行时接口规范 | gRPC + Protocol Buffers |
| **解耦** | kubelet 不依赖特定运行时 | CRI Shim 抽象层 |
| **可扩展** | 支持多种运行时接入 | 接口定义 + 插件机制 |
| **简化** | 减少 kubelet 运行时相关代码 | 通用接口调用 |

### 1.3 CRI 接口组成

```mermaid
graph TB
    subgraph "CRI 接口定义"
        RS[RuntimeService<br/>运行时服务]
        IS[ImageService<br/>镜像服务]
    end
    
    subgraph "RuntimeService 方法"
        RS1[Pod 沙箱管理]
        RS2[容器生命周期]
        RS3[容器查询]
        RS4[执行交互]
        RS5[网络管理]
    end
    
    subgraph "ImageService 方法"
        IS1[镜像拉取/删除]
        IS2[镜像查询]
        IS3[镜像存储信息]
    end
    
    RS --> RS1
    RS --> RS2
    RS --> RS3
    RS --> RS4
    RS --> RS5
    
    IS --> IS1
    IS --> IS2
    IS --> IS3
    
    style RS fill:#e3f2fd
    style IS fill:#fff3e0
```

---

## 2. 接口定义

### 2.1 gRPC 服务定义

CRI 使用 gRPC 定义两个核心服务：

```protobuf
// ============================================================
// RuntimeService - 容器和 Pod 生命周期管理
// ============================================================

service RuntimeService {
    // Pod 沙箱管理
    rpc RunPodSandbox(RunPodSandboxRequest) returns (RunPodSandboxResponse) {}
    rpc StopPodSandbox(StopPodSandboxRequest) returns (StopPodSandboxResponse) {}
    rpc RemovePodSandbox(RemovePodSandboxRequest) returns (RemovePodSandboxResponse) {}
    rpc ListPodSandbox(ListPodSandboxRequest) returns (ListPodSandboxResponse) {}
    rpc PodSandboxStatus(PodSandboxStatusRequest) returns (PodSandboxStatusResponse) {}

    // 容器生命周期
    rpc CreateContainer(CreateContainerRequest) returns (CreateContainerResponse) {}
    rpc StartContainer(StartContainerRequest) returns (StartContainerResponse) {}
    rpc StopContainer(StopContainerRequest) returns (StopContainerResponse) {}
    rpc RemoveContainer(RemoveContainerRequest) returns (RemoveContainerResponse) {}
    rpc ListContainers(ListContainersRequest) returns (ListContainersResponse) {}
    rpc ContainerStatus(ContainerStatusRequest) returns (ContainerStatusResponse) {}

    // 执行交互
    rpc ExecSync(ExecSyncRequest) returns (ExecSyncResponse) {}
    rpc Exec(ExecRequest) returns (ExecResponse) {}
    rpc Attach(AttachRequest) returns (AttachResponse) {}
    rpc PortForward(PortForwardRequest) returns (PortForwardResponse) {}

    // 其他
    rpc UpdateContainerResources(UpdateContainerResourcesRequest) 
        returns (UpdateContainerResourcesResponse) {}
    rpc Version(VersionRequest) returns (VersionResponse) {}
    rpc Status(StatusRequest) returns (StatusResponse) {}
}

// ============================================================
// ImageService - 镜像管理
// ============================================================

service ImageService {
    rpc ListImages(ListImagesRequest) returns (ListImagesResponse) {}
    rpc ImageStatus(ImageStatusRequest) returns (ImageStatusResponse) {}
    rpc PullImage(PullImageRequest) returns (PullImageResponse) {}
    rpc RemoveImage(RemoveImageRequest) returns (RemoveImageResponse) {}
    rpc ImageFsInfo(ImageFsInfoRequest) returns (ImageFsInfoResponse) {}
}
```

### 2.2 接口版本演进

```mermaid
graph LR
    subgraph "版本演进"
        V1[v1alpha1<br/>Kubernetes 1.5]
        V2[v1<br/>Kubernetes 1.23]
    end
    
    V1 -->|"升级"| V2
    
    style V1 fill:#ffcdd2
    style V2 fill:#c8e6c9
```

| 版本 | Kubernetes 版本 | 变化 |
|------|-----------------|------|
| **v1alpha1** | 1.5 - 1.22 | 初始版本，部分接口不稳定 |
| **v1** | 1.23+ | 稳定版本，接口冻结 |

### 2.3 Socket 通信

kubelet 通过 Unix Socket 与运行时通信：

| 运行时 | Socket 路径 |
|--------|-------------|
| containerd | `/run/containerd/containerd.sock` |
| CRI-O | `/run/crio/crio.sock` |
| cri-dockerd | `/run/cri-dockerd.sock` |

```bash
# ============================================================
# 查看 Socket 文件
# ============================================================

ls -la /run/containerd/containerd.sock
ls -la /run/crio/crio.sock

# ============================================================
# 使用 grpcurl 直接调用 CRI
# ============================================================

grpcurl -plaintext unix:///run/containerd/containerd.sock \
    runtime.v1.RuntimeService/Version
```

---

## 3. RuntimeService 接口

### 3.1 Pod 沙箱管理

Pod 沙箱（Pod Sandbox）是 Pod 内所有容器共享的基础环境，包括：
- 共享的网络命名空间
- 共享的 IPC 命名空间
- 共享的 PID 命名空间（可选）
- Pod 级别的 cgroup

```mermaid
graph TB
    subgraph "Pod 沙箱结构"
        PS[Pod Sandbox<br/>共享命名空间]
        
        subgraph "共享资源"
            NET[网络命名空间<br/>Pod IP]
            IPC[IPC 命名空间<br/>共享内存]
            PID[PID 命名空间<br/>进程共享<br/>可选]
            CG[cgroup<br/>资源限制]
        end
        
        subgraph "容器"
            C1[容器1]
            C2[容器2]
            C3[容器3]
        end
    end
    
    PS --> NET
    PS --> IPC
    PS --> PID
    PS --> CG
    
    NET --> C1
    NET --> C2
    NET --> C3
    
    IPC --> C1
    IPC --> C2
    
    style PS fill:#fff3e0
    style NET fill:#e3f2fd
    style IPC fill:#e3f2fd
    style PID fill:#e3f2fd
    style CG fill:#fce4ec
```

**核心方法：**

| 方法 | 功能 | 调用时机 |
|------|------|----------|
| `RunPodSandbox` | 创建 Pod 沙箱 | Pod 创建时首先调用 |
| `StopPodSandbox` | 停止 Pod 沙箱 | Pod 停止时调用 |
| `RemovePodSandbox` | 删除 Pod 沙箱 | Pod 删除时调用 |
| `ListPodSandbox` | 列出 Pod 沙箱 | kubelet 定期查询 |
| `PodSandboxStatus` | 查询沙箱状态 | 状态同步 |

### 3.2 RunPodSandbox 详解

```protobuf
// ============================================================
// RunPodSandboxRequest - 创建 Pod 氛围请求
// ============================================================

message RunPodSandboxRequest {
    // Pod 配置
    PodSandboxConfig config = 1;
    
    // 运行时处理器名称（可选）
    string runtime_handler = 2;
}

message RunPodSandboxResponse {
    // Pod 沙箱 ID
    string pod_sandbox_id = 1;
}

// ============================================================
// PodSandboxConfig - Pod 沙箱配置
// ============================================================

message PodSandboxConfig {
    // Pod 元数据
    PodSandboxMetadata metadata = 1;
    
    // 主机名
    string hostname = 2;
    
    // 日志目录
    string log_directory = 3;
    
    // DNS 配置
    DNSConfig dns_config = 4;
    
    // 端口映射
    repeated PortMapping port_mappings = 5;
    
    // 资源限制
    LinuxPodSandboxConfig linux = 6;
    
    // Labels 和 Annotations
    map<string, string> labels = 7;
    map<string, string> annotations = 8;
}
```

**调用示例：**

```mermaid
sequenceDiagram
    participant KL as kubelet
    participant CRI as CRI Shim
    participant RT as 容器运行时
    participant NET as 网络插件
    
    Note over KL,RT: RunPodSandbox 调用流程
    
    KL->>CRI: RunPodSandbox(config)
    
    Note over CRI: 1. 创建沙箱容器
    CRI->>RT: 创建 pause 容器
    
    Note over CRI: 2. 设置命名空间
    RT-->>CRI: 容器 ID
    
    Note over KL,NET: 3. 配置网络
    KL->>NET: SetUpPod(pod)
    NET->>RT: 配置网络接口
    NET-->>KL: 网络就绪
    
    Note over KL,RT: 4. 返回沙箱 ID
    RT-->>CRI: 沙箱就绪
    CRI-->>KL: PodSandboxId
```

### 3.3 容器生命周期管理

```mermaid
stateDiagram-v2
    [*] --> Created: CreateContainer
    
    Created --> Running: StartContainer
    Running --> Stopped: StopContainer
    Stopped --> [*]: RemoveContainer
    
    Created --> [*]: RemoveContainer
    
    Running --> Exited: 容器退出
    Exited --> [*]: RemoveContainer
    
    note right of Created: 容器已创建<br/>但未启动
    note right of Running: 容器正在运行
    note right of Stopped: 容器已停止
    note right of Exited: 容器异常退出
```

**核心方法：**

| 方法 | 功能 | 调用时机 |
|------|------|----------|
| `CreateContainer` | 创建容器 | Pod 沙箱创建后 |
| `StartContainer` | 启动容器 | 容器创建后 |
| `StopContainer` | 停止容器 | Pod 停止或容器异常 |
| `RemoveContainer` | 删除容器 | Pod 删除时 |
| `ListContainers` | 列出容器 | kubelet 定期查询 |
| `ContainerStatus` | 查询容器状态 | 状态同步 |

### 3.4 CreateContainer 详解

```protobuf
// ============================================================
// CreateContainerRequest - 创建容器请求
// ============================================================

message CreateContainerRequest {
    // Pod 沙箱 ID
    string pod_sandbox_id = 1;
    
    // 容器配置
    ContainerConfig config = 2;
    
    // 沙箱配置（用于继承）
    PodSandboxConfig sandbox_config = 3;
}

message CreateContainerResponse {
    // 容器 ID
    string container_id = 1;
}

// ============================================================
// ContainerConfig - 容器配置
// ============================================================

message ContainerConfig {
    // 容器元数据
    ContainerMetadata metadata = 1;
    
    // 镜像
    ImageSpec image = 2;
    
    // 命令
    repeated string command = 3;
    repeated string args = 4;
    
    // 工作目录
    string working_dir = 5;
    
    // 环境变量
    repeated KeyValue envs = 6;
    
    // 挂载点
    repeated Mount mounts = 7;
    
    // 设备
    repeated Device devices = 8;
    
    // 资源限制
    LinuxContainerResources resources = 9;
    
    // Labels 和 Annotations
    map<string, string> labels = 10;
    map<string, string> annotations = 11;
    
    // 日志路径
    string log_path = 12;
    
    // 是否 stdin/stdout
    bool stdin = 13;
    bool stdin_once = 14;
    bool tty = 15;
}
```

---

## 4. ImageService 接口

### 4.1 镜像管理方法

```mermaid
graph TB
    subgraph "ImageService 方法"
        PI[PullImage<br/>拉取镜像]
        RI[RemoveImage<br/>删除镜像]
        LI[ListImages<br/>列出镜像]
        IS[ImageStatus<br/>镜像状态]
        IF[ImageFsInfo<br/>镜像存储信息]
    end
    
    subgraph "调用时机"
        T1[Pod 创建前<br/>检查镜像是否存在]
        T2[镜像不存在<br/>自动拉取]
        T3[镜像清理<br/>定期删除未使用镜像]
        T4[状态同步<br/>定期查询镜像列表]
    end
    
    PI --> T2
    RI --> T3
    LI --> T4
    IS --> T1
    IF --> T4
    
    style PI fill:#fff3e0
    style RI fill:#ffcdd2
    style LI fill:#e3f2fd
    style IS fill:#c8e6c9
    style IF fill:#fce4ec
```

### 4.2 PullImage 详解

```protobuf
// ============================================================
// PullImageRequest - 拉取镜像请求
// ============================================================

message PullImageRequest {
    // 镜像规范
    ImageSpec image = 1;
    
    // 认证信息
    AuthConfig auth = 2;
    
    // 沙箱配置（用于继承）
    PodSandboxConfig sandbox_config = 3;
}

message PullImageResponse {
    // 镜像引用
    string image_ref = 1;
}

// ============================================================
// ImageSpec - 镜像规范
// ============================================================

message ImageSpec {
    // 镜像名称（如 nginx:latest）
    string image = 1;
    
    // Annotations
    map<string, string> annotations = 2;
}
```

**拉取流程：**

```mermaid
sequenceDiagram
    participant KL as kubelet
    participant CRI as CRI Shim
    participant REG as 镜像仓库
    participant STORE as 镜像存储
    
    Note over KL,STORE: 镜像拉取流程
    
    KL->>CRI: PullImage(image, auth)
    
    Note over CRI: 1. 解析镜像名称
    CRI->>CRI: 解析 registry/tag
    
    Note over CRI,REG: 2. 请求镜像仓库
    CRI->>REG: GET /v2/<name>/manifests/<ref>
    REG-->>CRI: Manifest
    
    Note over CRI,REG: 3. 拉取镜像层
    loop 每个层
        CRI->>REG: GET /v2/<name>/blobs/<digest>
        REG-->>CRI: Layer Data
        CRI->>STORE: 存储层数据
    end
    
    Note over CRI,STORE: 4. 组装镜像
    CRI->>STORE: 创建镜像快照
    
    Note over KL,CRI: 5. 返回镜像引用
    CRI-->>KL: ImageRef
```

### 4.3 镜像存储结构

```mermaid
graph TB
    subgraph "镜像存储架构"
        REG[镜像仓库]
        
        subgraph "本地存储"
            CS[Content Store<br/>镜像层存储]
            SN[Snapshotter<br/>镜像快照]
        end
        
        subgraph "镜像组成"
            MAN[Manifest<br/>镜像清单]
            CFG[Config<br/>镜像配置]
            L1[Layer 1<br/>基础层]
            L2[Layer 2<br/>应用层]
            L3[Layer 3<br/>配置层]
        end
    end
    
    REG -->|"拉取"| CS
    CS --> L1
    CS --> L2
    CS --> L3
    
    MAN --> CFG
    MAN --> L1
    MAN --> L2
    MAN --> L3
    
    SN -->|"组装"| L1
    SN -->|"组装"| L2
    SN -->|"组装"| L3
    
    style REG fill:#e3f2fd
    style CS fill:#fff3e0
    style SN fill:#c8e6c9
```

---

## 5. Pod 生命周期管理

### 5.1 Pod 创建完整流程

```mermaid
sequenceDiagram
    participant API as API Server
    participant KL as kubelet
    participant CRI as CRI Shim
    participant NET as 网络插件
    participant VOL as 卷插件
    
    Note over API,KL: 1. 接收 Pod 创建请求
    API->>KL: Pod 创建事件
    
    Note over KL: 2. 预检查
    KL->>KL: 检查资源、调度约束
    
    Note over KL,CRI: 3. 镜像准备
    KL->>CRI: ImageStatus(image)
    alt 镜像不存在
        KL->>CRI: PullImage(image)
        CRI-->>KL: ImageRef
    end
    
    Note over KL,VOL: 4. 卷挂载
    KL->>VOL: MountVolumes(pod)
    VOL-->>KL: 挂载完成
    
    Note over KL,CRI: 5. 创建 Pod 沙箱
    KL->>CRI: RunPodSandbox(config)
    CRI-->>KL: PodSandboxId
    
    Note over KL,NET: 6. 配置网络
    KL->>NET: SetUpPod(pod)
    NET-->>KL: 网络就绪
    
    Note over KL,CRI: 7. 创建容器
    loop 每个容器
        KL->>CRI: CreateContainer(sandboxId, config)
        CRI-->>KL: ContainerId
    end
    
    Note over KL,CRI: 8. 启动容器
    loop 每个容器
        KL->>CRI: StartContainer(containerId)
        CRI-->>KL: 容器运行
    end
    
    Note over KL,CRI: 9. 后续处理
    KL->>CRI: ContainerStatus(containerId)
    CRI-->>KL: 状态更新
    
    KL-->>API: Pod Running
```

### 5.2 Pod 删除流程

```mermaid
sequenceDiagram
    participant API as API Server
    participant KL as kubelet
    participant CRI as CRI Shim
    participant NET as 网络插件
    participant VOL as 卷插件
    
    Note over API,KL: 1. 接收删除请求
    API->>KL: Pod 删除事件
    
    Note over KL,CRI: 2. 停止容器
    loop 每个容器
        KL->>CRI: StopContainer(containerId, timeout)
        CRI-->>KL: 容器停止
        KL->>CRI: RemoveContainer(containerId)
        CRI-->>KL: 容器删除
    end
    
    Note over KL,NET: 3. 清理网络
    KL->>NET: TeardownPod(pod)
    NET-->>KL: 网络清理
    
    Note over KL,CRI: 4. 删除 Pod 沙箱
    KL->>CRI: StopPodSandbox(sandboxId)
    CRI-->>KL: 沙箱停止
    KL->>CRI: RemovePodSandbox(sandboxId)
    CRI-->>KL: 沙箱删除
    
    Note over KL,VOL: 5. 清理卷
    KL->>VOL: UnmountVolumes(pod)
    VOL-->>KL: 卷清理
    
    KL-->>API: Pod Deleted
```

### 5.3 Pod 沙箱与容器关系

```mermaid
graph TB
    subgraph "Pod 结构"
        PS[Pod Sandbox<br/>pause 容器]
        
        subgraph "容器进程"
            C1[容器1<br/>nginx]
            C2[容器2<br/>app]
            C3[容器3<br/>sidecar]
        end
        
        subgraph "共享资源"
            IP[Pod IP<br/>10.0.0.5]
            NET[网络命名空间]
            IPC[IPC 命名空间]
        end
    end
    
    PS --> IP
    IP --> NET
    NET --> C1
    NET --> C2
    NET --> C3
    
    IPC --> C1
    IPC --> C2
    
    style PS fill:#fff3e0
    style IP fill:#e3f2fd
    style NET fill:#e3f2fd
    style IPC fill:#e3f2fd
```

---

## 6. 容器生命周期管理

### 6.1 容器状态

```mermaid
stateDiagram-v2
    [*] --> Created: CreateContainer
    
    Created --> Running: StartContainer
    Created --> [*]: RemoveContainer
    
    Running --> Exited: 正常退出 (exit 0)
    Running --> Failed: 异常退出 (exit != 0)
    Running --> Stopped: StopContainer
    
    Exited --> Running: 重启策略 Restart
    Exited --> [*]: 重启策略 Never
    
    Failed --> Running: 重启策略 Always/OnFailure
    Failed --> [*]: 重启策略 Never
    
    Stopped --> [*]: RemoveContainer
    
    note right of Created: 容器已创建<br/>等待启动
    note right of Running: 容器正在运行<br/>处理请求
    note right of Exited: 容器正常退出<br/>exit code = 0
    note right of Failed: 容器异常退出<br/>exit code != 0
```

### 6.2 容器配置详解

```protobuf
// ============================================================
// LinuxContainerResources - Linux 容器资源限制
// ============================================================

message LinuxContainerResources {
    // CPU 限制
    int64 cpu_period = 1;      // CFS 周期（微秒）
    int64 cpu_quota = 2;       // CFS 配额（微秒）
    int64 cpu_shares = 3;      // CPU 权重
    int64 cpu_millicores = 4;  // CPU 核心数（毫核）
    
    // 内存限制
    int64 memory_limit_in_bytes = 5;
    int64 memory_swap_limit_in_bytes = 6;
    
    // OOM 控制
    int64 oom_score_adj = 7;   // OOM 分数调整
    
    // HugePages
    map<string, int64> hugepage_limits = 8;
    
    // 统一的资源限制（v1 新增）
    map<string, string> unified = 9;
}
```

### 6.3 挂载配置

```protobuf
// ============================================================
// Mount - 挂载配置
// ============================================================

message Mount {
    // 容器内路径
    string container_path = 1;
    
    // 主机路径
    string host_path = 2;
    
    // 只读挂载
    bool readonly = 3;
    
    // SELinux 重新标记
    string selinux_relabel = 4;
    
    // 传播模式
    MountPropagation propagation = 5;
}

enum MountPropagation {
    PROPAGATION_PRIVATE = 0;     // 私有挂载
    PROPAGATION_HOST_TO_CONTAINER = 1;  // 主机到容器传播
    PROPAGATION_BIDIRECTIONAL = 2;      // 双向传播
}
```

---

## 7. crictl 工具详解

### 7.1 crictl 概述

crictl 是 CRI 的命令行诊断工具，用于直接与容器运行时交互。

```mermaid
graph LR
    subgraph "crictl 位置"
        CLI[crictl]
        CRI[CRI Socket]
        KL[kubelet]
    end
    
    CLI -->|"gRPC"| CRI
    KL -->|"gRPC"| CRI
    
    style CLI fill:#fff3e0
    style CRI fill:#c8e6c9
    style KL fill:#e3f2fd
```

### 7.2 配置 crictl

```bash
# ============================================================
# crictl 配置
# ============================================================

# 配置运行时 socket
cat > /etc/crictl.yaml <<EOF
runtime-endpoint: unix:///run/containerd/containerd.sock
image-endpoint: unix:///run/containerd/containerd.sock
timeout: 10
debug: false
EOF

# 或者直接指定
crictl --runtime-endpoint unix:///run/containerd/containerd.sock pods
```

### 7.3 常用命令清单

```bash
# ============================================================
# Pod 相关命令
# ============================================================

# 列出 Pod
crictl pods

# 列出指定命名空间的 Pod
crictl pods --namespace kube-system

# 查看 Pod 状态
crictl inspectp <pod-id>

# 创建 Pod（测试）
crictl runp pod-config.json

# ============================================================
# 容器相关命令
# ============================================================

# 列出容器
crictl ps
crictl ps -a  # 包含已停止的

# 查看容器状态
crictl inspect <container-id>

# 查看容器日志
crictl logs <container-id>
crictl logs -f <container-id>  # 实时查看

# 在容器中执行命令
crictl exec <container-id> ls /
crictl exec -it <container-id> sh

# 停止容器
crictl stop <container-id>

# 删除容器
crictl rm <container-id>

# ============================================================
# 镜像相关命令
# ============================================================

# 列出镜像
crictl images

# 拉取镜像
crictl pull nginx:latest

# 删除镜像
crictl rmi nginx:latest

# 查看镜像详情
crictl inspecti nginx:latest

# ============================================================
# 运行时信息命令
# ============================================================

# 查看运行时版本
crictl version

# 查看运行时状态
crictl info

# 查看 ImageFS 信息
crictl imagefsinfo
```

### 7.4 实用诊断场景

**场景 1：Pod 卡在 ContainerCreating**

```bash
# ============================================================
# 诊断 ContainerCreating 问题
# ============================================================

# 1. 查看 Pod 状态
crictl pods | grep <pod-name>

# 2. 查看容器状态
crictl ps -a | grep <pod-id>

# 3. 检查镜像是否存在
crictl images | grep <image-name>

# 4. 检查运行时状态
crictl info

# 5. 查看运行时日志
journalctl -u containerd -f --no-pager
```

**场景 2：容器频繁重启**

```bash
# ============================================================
# 诊断容器重启问题
# ============================================================

# 1. 查看容器历史日志
crictl logs <container-id> --tail 100

# 2. 检查容器退出原因
crictl inspect <container-id>

# 3. 检查资源限制
crictl inspect <container-id> | grep -A 20 "resources"

# 4. 检查 OOM 事件
dmesg | grep -i "oom"
```

---

## 8. 最佳实践

### 8.1 运行时选择建议

```mermaid
graph TB
    subgraph "选择建议"
        PROD[生产环境<br/>推荐 containerd]
        DEV[开发环境<br/>可选 Docker]
        SEC[安全敏感<br/>选择 Kata/gVisor]
        K8S[纯 K8s 环境<br/>可选 CRI-O]
    end
    
    style PROD fill:#c8e6c9
    style DEV fill:#fff3e0
    style SEC fill:#e3f2fd
    style K8S fill:#c8e6c9
```

| 场景 | 推荐运行时 | 原因 |
|------|------------|------|
| **生产环境** | containerd | 稳定、高效、社区支持强 |
| **纯 K8s 环境** | CRI-O | 配置简单、专为 K8s 设计 |
| **开发环境** | Docker | 工具链完善、调试方便 |
| **安全敏感** | Kata/gVisor | 强隔离、安全防护 |

### 8.2 配置优化建议

| 配置项 | 建议 | 说明 |
|--------|------|------|
| **Snapshotter** | overlayfs | 性能最佳，生产推荐 |
| **镜像清理** | 定期清理未使用镜像 | 减少磁盘占用 |
| **日志配置** | 设置最大日志大小 | 防止日志占用过多空间 |
| **资源限制** | 合理设置默认限制 | 防止资源耗尽 |

### 8.3 排查问题建议

```mermaid
flowchart TB
    P[发现问题]
    
    subgraph "排查步骤"
        S1[检查 Pod 状态<br/>crictl pods]
        S2[检查容器状态<br/>crictl ps -a]
        S3[查看容器日志<br/>crictl logs]
        S4[检查运行时日志<br/>journalctl]
        S5[检查系统日志<br/>dmesg]
    end
    
    P --> S1 --> S2 --> S3 --> S4 --> S5
    
    style P fill:#ffcdd2
    style S1 fill:#e3f2fd
    style S2 fill:#e3f2fd
    style S3 fill:#fff3e0
    style S4 fill:#fff3e0
    style S5 fill:#c8e6c9
```

---

## 附录

### A. 接口方法速查表

| 方法 | 功能 | 所属服务 |
|------|------|----------|
| RunPodSandbox | 创建 Pod 沙箱 | RuntimeService |
| StopPodSandbox | 停止 Pod 沙箱 | RuntimeService |
| RemovePodSandbox | 删除 Pod 沙箱 | RuntimeService |
| CreateContainer | 创建容器 | RuntimeService |
| StartContainer | 启动容器 | RuntimeService |
| StopContainer | 停止容器 | RuntimeService |
| RemoveContainer | 删除容器 | RuntimeService |
| PullImage | 拉取镜像 | ImageService |
| RemoveImage | 删除镜像 | ImageService |
| ListImages | 列出镜像 | ImageService |

### B. 参考资料

| 资源 | 链接 |
|------|------|
| CRI API 定义 | https://github.com/kubernetes/cri-api |
| containerd 文档 | https://containerd.io/docs/ |
| CRI-O 文档 | https://cri-o.io/ |
| crictl 工具 | https://github.com/kubernetes-sigs/cri-tools |

---

> 继续学习：了解运行时接口后，推荐阅读 [runtime-deployment.md](runtime-deployment.md) 学习具体运行时的部署和配置。