// Package device 提供设备发现和管理的 Mock 实现
// 实际生产环境应对接真实的硬件发现机制（如 NVIDIA NVML、DCGM 等）
package device

// ============================================================
// 设备数据结构
// ============================================================

// Device 表示一个可分配的硬件设备
type Device struct {
	// ID 设备唯一标识符（用于 Kubernetes 分配）
	ID string

	// Health 设备健康状态
	Health string // "Healthy" 或 "Unhealthy"

	// Topology 设备拓扑信息（可选）
	Topology *TopologyInfo

	// Properties 设备属性（用于 Allocate 时配置）
	Properties map[string]string
}

// TopologyInfo 设备拓扑信息
type TopologyInfo struct {
	NUMANode   int    // NUMA 节点 ID
	PCIPath    string // PCI 设备路径
	GPUIndex   int    // GPU 累引（同一节点上的顺序）
}

// ============================================================
// Mock 设备发现函数
// ============================================================

// DiscoverDevices 发现节点上的所有 GPU 设备
// 生产环境示例：
//   - NVIDIA GPU: 使用 NVML 库查询设备列表
//   - 其他硬件: 对应厂商 SDK
func DiscoverDevices() []Device {
	// Mock 实现：返回模拟的 GPU 设备列表
	return []Device{
		{
			ID:     "gpu-0000:08:00.0",
			Health: "Healthy",
			Topology: &TopologyInfo{
				NUMANode: 0,
				PCIPath:  "/sys/bus/pci/devices/0000:08:00.0",
				GPUIndex: 0,
			},
			Properties: map[string]string{
				"uuid":      "GPU-abc123-0001",
				"model":     "NVIDIA-A100-SXM4-80GB",
				"memory":    "81920", // MiB
				"driver":    "535.129.03",
				"cuda":      "12.2",
			},
		},
		{
			ID:     "gpu-0000:09:00.0",
			Health: "Healthy",
			Topology: &TopologyInfo{
				NUMANode: 1,
				PCIPath:  "/sys/bus/pci/devices/0000:09:00.0",
				GPUIndex: 1,
			},
			Properties: map[string]string{
				"uuid":      "GPU-abc123-0002",
				"model":     "NVIDIA-A100-SXM4-80GB",
				"memory":    "81920",
				"driver":    "535.129.03",
				"cuda":      "12.2",
			},
		},
	}
}

// GetDriverPaths 获取驱动库挂载路径
func GetDriverPaths() []string {
	// NVIDIA 驱动默认挂载路径
	return []string{
		"/usr/lib/x86_64-linux-gnu/libcuda.so",
		"/usr/lib/x86_64-linux-gnu/libcudart.so",
		"/usr/lib/x86_64-linux-gnu/libnvidia-ml.so",
		"/usr/local/cuda/lib64",
	}
}

// GetDeviceFiles 获取设备文件路径
func GetDeviceFiles(deviceID string) []string {
	// NVIDIA 设备文件
	return []string{
		"/dev/nvidia0",      // GPU 控制设备
		"/dev/nvidiactl",    // NVIDIA 控制设备
		"/dev/nvidia-uvm",   // Unified Virtual Memory
		"/dev/nvidia-uvm-tools",
	}
}

// GetCUDABinaries 获取 CUDA 工具路径
func GetCUDABinaries() []string {
	return []string{
		"/usr/local/cuda/bin/nvidia-smi",
		"/usr/local/cuda/bin/nvcc",
	}
}