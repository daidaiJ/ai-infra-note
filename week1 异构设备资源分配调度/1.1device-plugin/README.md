# Kubernetes Device Plugin POC

一个 Device Plugin 原型实现，展示如何在设备分配时挂载驱动、设置环境变量和映射设备文件。

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **完整学习文档** - Allocate 接口详解、注入能力说明 |
| [pkg/device/device.go](pkg/device/device.go) | 设备发现 Mock 实现 |
| [pkg/plugin/plugin.go](pkg/plugin/plugin.go) | Device Plugin 核心接口实现 |
| [cmd/plugin/main.go](cmd/plugin/main.go) | 入口程序 |
| [manifests/daemonset.yaml](manifests/daemonset.yaml) | DaemonSet 部署清单 |

## Allocate 接口能力

```
┌─────────────────────────────────────────────────────────────┐
│                 ContainerAllocateResponse                    │
├─────────────────────────────────────────────────────────────┤
│  Envs        → 注入环境变量 (NVIDIA_VISIBLE_DEVICES 等)      │
│  Mounts      → 挂载驱动库、CUDA 工具                          │
│  Devices     → 映射 /dev/nvidia* 设备文件                    │
│  Annotations → 容器级别元数据                                 │
└─────────────────────────────────────────────────────────────┘
```

## 项目结构

```
device-plugin/
├── cmd/plugin/main.go      # 入口：启动 gRPC 服务、注册插件
├── pkg/
│   ├── device/device.go    # 设备发现 Mock
│   └── plugin/plugin.go    # Allocate 接口实现（核心）
├── manifests/              # DaemonSet + Pod 示例
└── docs/README.md          # 详细学习文档
```

## 核心代码示意

```go
// Allocate 接口：设备分配时修改 Pod 声明
func (p *GPUDevicePlugin) Allocate(ctx, req) (*AllocateResponse, error) {
    for _, containerReq := range req.ContainerRequests {
        resp := &ContainerAllocateResponse{}
        
        // 1. 环境变量
        resp.Envs["NVIDIA_VISIBLE_DEVICES"] = deviceID
        resp.Envs["CUDA_VERSION"] = "12.2"
        
        // 2. 挂载驱动库
        resp.Mounts = []Mount{
            {HostPath: "/usr/lib/x86_64-linux-gnu", ContainerPath: "...", ReadOnly: true},
        }
        
        // 3. 设备文件
        resp.Devices = []DeviceSpec{
            {HostPath: "/dev/nvidia0", ContainerPath: "/dev/nvidia0", Permissions: "rwm"},
        }
        
        // 4. Annotations
        resp.Annotations["device-uuid"] = uuid
    }
    return response, nil
}
```

详见 **[docs/README.md](docs/README.md)** 获取 Allocate 接口完整能力说明。