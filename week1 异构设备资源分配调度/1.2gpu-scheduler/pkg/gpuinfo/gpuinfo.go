// Package gpuinfo 提供 GPU 信息获取的抽象接口
// 实际生产环境中，可通过 DCGM Exporter、NVIDIA SMI 或 Prometheus 等获取真实数据
package gpuinfo

import (
	"k8s.io/api/core/v1"
)

// GPUInfo 表示单个 GPU 设备的信息
type GPUInfo struct {
	// UUID 设备唯一标识符
	UUID string
	// NodeName 所在节点名称
	NodeName string
	// Utilization 当前利用率 (0-100)
	Utilization float64
	// MemoryUsed 已用显存 (MiB)
	MemoryUsed int64
	// MemoryTotal 总显存 (MiB)
	MemoryTotal int64
	// IsHealthy 设备健康状态
	IsHealthy bool
}

// ============================================
// Mock 函数：示意性实现，表达关键语义
// ============================================

// GetNodeGPUUtilization 获取节点的 GPU 平均利用率
// 生产环境可从 Prometheus 查询: avg(DCGM_FI_DEV_GPU_UTIL{node=~"node-name"})
func GetNodeGPUUtilization(nodeName string) float64 {
	// Mock 实现：返回模拟数据
	// 实际应从监控系统获取实时数据
	mockData := map[string]float64{
		"node-gpu-01": 25.5,
		"node-gpu-02": 78.3,
		"node-gpu-03": 45.0,
	}
	return mockData[nodeName]
}

// GetNodeByDeviceUUID 根据设备 UUID 查找所在节点
// 生产环境可从 Node 资源的注解或 Device Plugin 暴露的资源中查询
func GetNodeByDeviceUUID(deviceUUID string) string {
	// Mock 实现：返回模拟节点名
	// 实际应查询 Kubernetes API 或设备管理服务
	mockMapping := map[string]string{
		"GPU-abc123-0001": "node-gpu-01",
		"GPU-abc123-0002": "node-gpu-01",
		"GPU-xyz789-0001": "node-gpu-02",
		"GPU-xyz789-0002": "node-gpu-02",
	}
	return mockMapping[deviceUUID]
}

// GetAllNodesGPUInfo 批量获取所有 GPU 节点的信息
func GetAllNodesGPUInfo() []GPUInfo {
	// Mock 实现：返回模拟的集群 GPU 状态
	return []GPUInfo{
		{UUID: "GPU-abc123-0001", NodeName: "node-gpu-01", Utilization: 20.0, MemoryTotal: 81920, MemoryUsed: 16384, IsHealthy: true},
		{UUID: "GPU-abc123-0002", NodeName: "node-gpu-01", Utilization: 31.0, MemoryTotal: 81920, MemoryUsed: 24576, IsHealthy: true},
		{UUID: "GPU-xyz789-0001", NodeName: "node-gpu-02", Utilization: 85.0, MemoryTotal: 81920, MemoryUsed: 65536, IsHealthy: true},
		{UUID: "GPU-xyz789-0002", NodeName: "node-gpu-02", Utilization: 71.6, MemoryTotal: 81920, MemoryUsed: 57344, IsHealthy: true},
	}
}