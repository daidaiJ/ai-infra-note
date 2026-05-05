// Package plugin 实现 Kubernetes Device Plugin 核心逻辑
// Device Plugin 通过 gRPC 与 kubelet 通信，负责设备发现和分配
package plugin

import (
	"context"
	"fmt"

	"github.com/example/device-plugin/pkg/device"
	"k8s.io/klog/v2"

	// Device Plugin gRPC 接口定义
	// 实际使用: k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1
)

// ============================================================
// Device Plugin 接口定义（核心）
// ============================================================
//
// Kubernetes Device Plugin 需实现以下 gRPC 服务：
//
// service DevicePlugin {
//   // 获取插件选项（可选功能支持）
//   rpc GetDevicePluginOptions(Empty) returns (DevicePluginOptions) {}
//
//   // 流式返回设备列表（核心）
//   rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}
//
//   // 设备分配时修改 Pod 声明（核心，本文件重点）
//   rpc Allocate(AllocateRequest) returns (AllocateResponse) {}
//
//   // 返回首选分配方案（可选）
//   rpc GetPreferredAllocation(PreferredAllocationRequest) returns (PreferredAllocationResponse) {}
//
//   // 容器启动前准备（可选）
//   rpc PreStartContainer(PreStartContainerRequest) returns (PreStartContainerResponse) {}
// }

// ============================================================
// 插件结构体
// ============================================================

type GPUDevicePlugin struct {
	// 设备列表（从 DiscoverDevices 获取）
	devices map[string]device.Device

	// gRPC 服务端
	server interface{} // 实际类型: *grpc.Server

	// 插件 socket 路径
	socketPath string

	// 资源名称（如 nvidia.com/gpu）
	resourceName string
}

// ============================================================
// GetDevicePluginOptions - 返回插件配置选项
// ============================================================
//
// 作用：告诉 kubelet 该插件支持哪些可选功能
//
func (p *GPUDevicePlugin) GetDevicePluginOptions(ctx context.Context, req *Empty) (*DevicePluginOptions, error) {
	return &DevicePluginOptions{
		// 是否支持 PreStartContainer 调用
		PreStartRequired: false,

		// 是否支持 GetPreferredAllocation 调用
		GetPreferredAllocationAvailable: true,
	}, nil
}

// ============================================================
// ListAndWatch - 设备发现与状态上报（核心接口）
// ============================================================
//
// 作用：持续向 kubelet 上报节点上的设备列表和健康状态
// kubelet 根据此信息更新 Node 资源的 Capacity
//
func (p *GPUDevicePlugin) ListAndWatch(req *Empty, stream ListAndWatchStream) error {
	klog.Info("启动设备发现与监控...")

	// === 步骤1: 发现设备 ===
	devices := device.DiscoverDevices()

	// === 步骤2: 构建响应并发送 ===
	resp := &ListAndWatchResponse{
		Devices: make([]Device, len(devices)),
	}

	for i, dev := range devices {
		resp.Devices[i] = Device{
			ID:     dev.ID,
			Health: dev.Health,
			// Topology 可选，用于 NUMA 感知调度
			Topology: &TopologyInfo{
				NUMANode: dev.Topology.NUMANode,
			},
		}
		p.devices[dev.ID] = dev
	}

	// 发送初始设备列表
	if err := stream.Send(resp); err != nil {
		return err
	}

	// === 步骤3: 持续监控设备健康状态 ===
	// 实际实现：定期检查设备状态，状态变化时发送更新
	for {
		// Mock: 模拟健康检查循环
		// 生产环境：对接硬件健康监控服务
		select {
		case <-stream.Context().Done():
			return nil
		}
	}
}

// ============================================================
// Allocate - 设备分配时的 Pod 声明修改（核心接口，重点！）
// ============================================================
//
// 【调用时机】
// 当 kubelet 创建 Pod 并分配设备时，调用此接口
// 在 Pod 创建之前，允许修改 Pod 的容器声明
//
// 【可修改的内容】（详见文档）
// 1. 环境变量 (Env) - 设置设备相关信息
// 2. 挂载点 (Mounts) - 挂载驱动库、CUDA 工具等
// 3. 设备文件 (Devices) - 添加 /dev/nvidia* 等设备
// 4. Annotations - 添加容器级别的元数据
//
func (p *GPUDevicePlugin) Allocate(ctx context.Context, req *AllocateRequest) (*AllocateResponse, error) {
	klog.InfoS("收到设备分配请求", "devices", req.ContainerRequests)

	// === AllocateResponse 结构 ===
	// 每个容器对应一个 ContainerAllocateResponse
	response := &AllocateResponse{
		ContainerResponses: make([]*ContainerAllocateResponse, 0),
	}

	// === 处理每个容器的设备分配请求 ===
	for _, containerReq := range req.ContainerRequests {

		// 容器级别的响应
		containerResp := &ContainerAllocateResponse{}

		// ========================================================
		// 1. 设置环境变量 (Env)
		// ========================================================
		// 作用：向容器注入设备相关配置信息
		//
		// 【常用环境变量】
		// - NVIDIA_VISIBLE_DEVICES: 可见的 GPU 设备 ID
		// - CUDA_VISIBLE_DEVICES: CUDA 可见设备
		// - NVIDIA_DRIVER_VERSION: 驱动版本
		// - GPU_UUID: 设备 UUID
		//
		containerResp.Envs = p.buildEnvironmentVariables(containerReq.DevicesIDs)

		// ========================================================
		// 2. 设置挂载点 (Mounts)
		// ========================================================
		// 作用：将宿主机的驱动库、CUDA 工具等挂载到容器
		//
		// 【常用挂载点】
		// - NVIDIA 驱动库: /usr/lib/x86_64-linux-gnu/libcuda*.so
		// - CUDA 库: /usr/local/cuda/lib64
		// - CUDA 工具: /usr/local/cuda/bin
		//
		containerResp.Mounts = p.buildMounts()

		// ========================================================
		// 3. 设置设备文件 (Devices)
		// ========================================================
		// 作用：将宿主机的设备文件映射到容器
		//
		// 【常用设备文件】
		// - /dev/nvidia0: GPU 控制设备
		// - /dev/nvidiactl: NVIDIA 控制设备
		// - /dev/nvidia-uvm: Unified Virtual Memory
		//
		containerResp.Devices = p.buildDevices(containerReq.DevicesIDs)

		// ========================================================
		// 4. 设置 Annotations（可选）
		// ========================================================
		// 作用：添加容器级别的元数据，供其他组件使用
		//
		// 【常用 Annotations】
		// - 设备分配信息记录
		// - 驱动版本标记
		//
		containerResp.Annotations = p.buildAnnotations(containerReq.DevicesIDs)

		response.ContainerResponses = append(response.ContainerResponses, containerResp)
	}

	return response, nil
}

// ============================================================
// Allocate 辅助方法实现
// ============================================================

// buildEnvironmentVariables 构建环境变量
func (p *GPUDevicePlugin) buildEnvironmentVariables(deviceIDs []string) map[string]string {
	envs := map[string]string{}

	// === 设备可见性环境变量 ===
	// NVIDIA_VISIBLE_DEVICES: NVIDIA 官方容器运行时使用
	// 格式: "GPU-abc123-0001,GPU-abc123-0002" 或 "0,1" (索引)
	visibleDevices := ""
	for i, id := range deviceIDs {
		if i > 0 {
			visibleDevices += ","
		}
		visibleDevices += id
	}
	envs["NVIDIA_VISIBLE_DEVICES"] = visibleDevices

	// === CUDA 可见设备 ===
	// CUDA_VISIBLE_DEVICES: CUDA 应用程序使用
	// 格式与 NVIDIA_VISIBLE_DEVICES 相同
	envs["CUDA_VISIBLE_DEVICES"] = visibleDevices

	// === 设备 UUID ===
	// 用于应用程序精确识别设备
	if len(deviceIDs) > 0 {
		if dev, ok := p.devices[deviceIDs[0]]; ok {
			envs["NVIDIA_DRIVER_VERSION"] = dev.Properties["driver"]
			envs["CUDA_VERSION"] = dev.Properties["cuda"]
			envs["GPU_UUID"] = dev.Properties["uuid"]
			envs["GPU_MEMORY_MIB"] = dev.Properties["memory"]
			envs["GPU_MODEL"] = dev.Properties["model"]
		}
	}

	// === NUMA 感知配置（可选）===
	// 帮助应用优化内存访问
	if len(deviceIDs) > 0 {
		if dev, ok := p.devices[deviceIDs[0]]; ok && dev.Topology != nil {
			envs["GPU_NUMA_NODE"] = fmt.Sprintf("%d", dev.Topology.NUMANode)
			envs["GPU_PCI_PATH"] = dev.Topology.PCIPath
		}
	}

	// === 自定义环境变量示例 ===
	// 可根据业务需求添加更多配置
	envs["NVIDIA_REQUIRE_CUDA"] = "cuda>=12.0"

	return envs
}

// buildMounts 构驱动和工具挂载点
func (p *GPUDevicePlugin) buildMounts() []Mount {
	mounts := []Mount{}

	// ========================================================
	// 挂载类型说明:
	//
	// Mount {
	//   HostPath      string  // 宿主机路径
	//   ContainerPath string  // 容器内路径
	//   ReadOnly      bool    // 是否只读
	// }
	//
	// 原则：
	// - 驱动库: 只读挂载，避免容器污染
	// - 工具目录: 可读可写（如需生成临时文件）
	// ========================================================

	// === 1. NVIDIA 驱动库挂载 ===
	// 挂载核心驱动共享库
	mounts = append(mounts, Mount{
		HostPath:      "/usr/lib/x86_64-linux-gnu",
		ContainerPath: "/usr/lib/x86_64-linux-gnu",
		ReadOnly:      true, // 驱动库只读
	})

	// === 2. CUDA 库挂载 ===
	// 挂载 CUDA 运行时和开发库
	mounts = append(mounts, Mount{
		HostPath:      "/usr/local/cuda/lib64",
		ContainerPath: "/usr/local/cuda/lib64",
		ReadOnly:      true,
	})

	// === 3. CUDA 工具挂载 ===
	// 挂载 nvidia-smi, nvcc 等工具
	mounts = append(mounts, Mount{
		HostPath:      "/usr/local/cuda/bin",
		ContainerPath: "/usr/local/cuda/bin",
		ReadOnly:      false, // 工具目录可写（某些工具需临时文件）
	})

	// === 4. NVIDIA 驱动目录（包含版本化库）===
	mounts = append(mounts, Mount{
		HostPath:      "/usr/lib/nvidia",
		ContainerPath: "/usr/lib/nvidia",
		ReadOnly:      true,
	})

	// === 5. CUDA include 目录（开发需要）===
	mounts = append(mounts, Mount{
		HostPath:      "/usr/local/cuda/include",
		ContainerPath: "/usr/local/cuda/include",
		ReadOnly:      true,
	})

	// === 6. NVIDIA 驱动配置文件 ===
	mounts = append(mounts, Mount{
		HostPath:      "/etc/nvidia",
		ContainerPath: "/etc/nvidia",
		ReadOnly:      true,
	})

	// === 7. ld.so.conf 配置（确保库可被发现）===
	mounts = append(mounts, Mount{
		HostPath:      "/usr/local/cuda/lib64/stubs",
		ContainerPath: "/usr/local/cuda/lib64/stubs",
		ReadOnly:      true,
	})

	return mounts
}

// buildDevices 构建设备文件映射
func (p *GPUDevicePlugin) buildDevices(deviceIDs []string) []DeviceSpec {
	devices := []DeviceSpec{}

	// ========================================================
	// DeviceSpec 结构:
	//
	// DeviceSpec {
	//   HostPath      string  // 宿主机设备路径
	//   ContainerPath string  // 容器内设备路径
	//   Permissions   string  // 访问权限: "r"(读), "w"(写), "m"(mmap)
	// }
	//
	// 说明：
	// - Permissions: GPU 设备通常需要 "rwm" 完全权限
	// ========================================================

	// === 1. NVIDIA 控制设备 ===
	// nvidiactl: NVIDIA 统一控制接口
	devices = append(devices, DeviceSpec{
		HostPath:      "/dev/nvidiactl",
		ContainerPath: "/dev/nvidiactl",
		Permissions:   "rwm", // 读、写、mmap
	})

	// === 2. NVIDIA UVM 设备 ===
	// nvidia-uvm: Unified Virtual Memory，跨 GPU 内存管理
	devices = append(devices, DeviceSpec{
		HostPath:      "/dev/nvidia-uvm",
		ContainerPath: "/dev/nvidia-uvm",
		Permissions:   "rwm",
	})

	// === 3. NVIDIA UVM Tools ===
	devices = append(devices, DeviceSpec{
		HostPath:      "/dev/nvidia-uvm-tools",
		ContainerPath: "/dev/nvidia-uvm-tools",
		Permissions:   "rwm",
	})

	// === 4. 分配的 GPU 设备 ===
	// 根据请求的设备 ID 添加对应的 /dev/nvidiaN
	for _, deviceID := range deviceIDs {
		dev, ok := p.devices[deviceID]
		if !ok {
			continue
		}

		// 根据 GPU 索引确定设备文件名
		deviceFileName := fmt.Sprintf("/dev/nvidia%d", dev.Topology.GPUIndex)

		devices = append(devices, DeviceSpec{
			HostPath:      deviceFileName,
			ContainerPath: deviceFileName,
			Permissions:   "rwm",
		})
	}

	// === 5. NVIDIA Modeset 设备（可选）===
	// 用于 GPU 显示功能
	devices = append(devices, DeviceSpec{
		HostPath:      "/dev/nvidia-modeset",
		ContainerPath: "/dev/nvidia-modeset",
		Permissions:   "rwm",
	})

	return devices
}

// buildAnnotations 构建容器 Annotations
func (p *GPUDevicePlugin) buildAnnotations(deviceIDs []string) map[string]string {
	annotations := map[string]string{}

	// ========================================================
	// Annotations 说明:
	//
	// Annotations 不是 Kubernetes Pod 的 annotations
	// 而是 ContainerAllocateResponse 的 annotations
	// 用于在容器级别添加元数据
	//
	// 用途：
	// - 记录设备分配信息，供日志/监控使用
	// - 传递给其他运行时组件（如 NVIDIA Container Runtime）
	// ========================================================

	if len(deviceIDs) > 0 {
		// 记录分配的设备 ID
		annotations["device-plugin.allocated-devices"] = deviceIDs[0]

		if dev, ok := p.devices[deviceIDs[0]]; ok {
			// 记录设备详细信息
			annotations["device-plugin.device-uuid"] = dev.Properties["uuid"]
			annotations["device-plugin.device-model"] = dev.Properties["model"]
			annotations["device-plugin.driver-version"] = dev.Properties["driver"]
			annotations["device-plugin.cuda-version"] = dev.Properties["cuda"]

			// NUMA 拓扑信息
			if dev.Topology != nil {
				annotations["device-plugin.numa-node"] = fmt.Sprintf("%d", dev.Topology.NUMANode)
			}
		}
	}

	return annotations
}

// ============================================================
// GetPreferredAllocation - 首选分配方案（可选接口）
// ============================================================
//
// 作用：在 DeviceClass 支持首选分配时，kubelet 会调用此接口
// 返回插件推荐的设备分配方案
//
func (p *GPUDevicePlugin) GetPreferredAllocation(ctx context.Context, req *PreferredAllocationRequest) (*PreferredAllocationResponse, error) {
	// 示例：优先分配同一 NUMA 节点的设备
	// 生产环境可根据业务需求实现更复杂的分配策略

	preferredIDs := []string{}

	// 简单实现：按健康状态和拓扑选择
	for _, dev := range p.devices {
		if dev.Health == "Healthy" {
			preferredIDs = append(preferredIDs, dev.ID)
		}
	}

	return &PreferredAllocationResponse{
		ContainerResponses: []*ContainerPreferredAllocationResponse{
			{
				PreferredAllocationIDs: preferredIDs,
			},
		},
	}, nil
}

// ============================================================
// PreStartContainer - 容器启动前准备（可选接口）
// ============================================================
//
// 作用：在容器启动前执行准备工作
// 如：设备初始化、权限设置、状态检查等
//
func (p *GPUDevicePlugin) PreStartContainer(ctx context.Context, req *PreStartContainerRequest) (*PreStartContainerResponse, error) {
	klog.InfoS("容器启动前准备", "devices", req.DevicesIDs)

	// 示例：检查设备是否就绪
	for _, id := range req.DevicesIDs {
		if dev, ok := p.devices[id]; ok {
			if dev.Health != "Healthy" {
				return nil, fmt.Errorf("设备 %s 不健康", id)
			}
		}
	}

	return &PreStartContainerResponse{
		StartAllowed: true,
	}, nil
}

// ============================================================
// 类型定义（对应 gRPC 消息类型，示意性）
// ============================================================

type Empty struct{}

type DevicePluginOptions struct {
	PreStartRequired                bool
	GetPreferredAllocationAvailable bool
}

type Device struct {
	ID       string
	Health   string
	Topology *TopologyInfo
}

type TopologyInfo struct {
	NUMANode int
}

type ListAndWatchResponse struct {
	Devices []Device
}

type AllocateRequest struct {
	ContainerRequests []*ContainerAllocateRequest
}

type ContainerAllocateRequest struct {
	DevicesIDs []string
}

type AllocateResponse struct {
	ContainerResponses []*ContainerAllocateResponse
}

type ContainerAllocateResponse struct {
	Envs        map[string]string // 环境变量
	Mounts      []Mount           // 挂载点
	Devices     []DeviceSpec      // 设备文件
	Annotations map[string]string // 元数据
}

type Mount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

type DeviceSpec struct {
	HostPath      string
	ContainerPath string
	Permissions   string
}

type PreferredAllocationRequest struct {
	ContainerRequests []*ContainerPreferredAllocationRequest
}

type ContainerPreferredAllocationRequest struct {
	AvailableDeviceIDs  []string
	MustIncludeDeviceIDs []string
}

type PreferredAllocationResponse struct {
	ContainerResponses []*ContainerPreferredAllocationResponse
}

type ContainerPreferredAllocationResponse struct {
	PreferredAllocationIDs []string
}

type PreStartContainerRequest struct {
	DevicesIDs []string
}

type PreStartContainerResponse struct {
	StartAllowed bool
}

// ListAndWatchStream 流式接口（示意）
type ListAndWatchStream interface {
	Send(*ListAndWatchResponse) error
	Context() context.Context
}