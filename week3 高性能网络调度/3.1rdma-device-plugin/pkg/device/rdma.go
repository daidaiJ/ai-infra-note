// Package device RDMA设备发现逻辑
package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ============================================================
// 结构体定义
// ============================================================

// DeviceInfo RDMA设备详细信息
type DeviceInfo struct {
	// ID 设备唯一标识（如 mlx5_0）
	ID string

	// Type 设备类型（IB或RoCE）
	Type string // "ib" 或 "roce"

	// Health 设备健康状态
	Health bool

	// Topology NUMA拓扑信息
	Topology *TopologyInfo
}

// TopologyInfo NUMA拓扑信息
type TopologyInfo struct {
	// NUMANode NUMA节点编号
	NUMANode int

	// PCIAddress PCI地址
	PCIAddress string
}

// ============================================================
// 设备发现
// ============================================================

// DiscoverDevices 发现节点上的RDMA设备
func DiscoverDevices() ([]*DeviceInfo, error) {
	// ============================================================
	// 读取sysfs设备目录
	// ============================================================
	devicesDir := "/sys/class/infiniband"

	entries, err := os.ReadDir(devicesDir)
	if err != nil {
		// 目录不存在表示无RDMA设备
		if os.IsNotExist(err) {
			return []*DeviceInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read devices directory: %v", err)
	}

	// ============================================================
	// 解析每个设备信息
	// ============================================================
	devices := make([]*DeviceInfo, 0, len(entries))

	for _, entry := range entries {
		deviceName := entry.Name()

		// === 步骤1: 获取设备类型 ===
		deviceType := getDeviceType(deviceName)

		// === 步骤2: 获取NUMA拓扑 ===
		topology := getDeviceTopology(deviceName)

		// === 步骤3: 检查健康状态 ===
		health := CheckDeviceHealth(deviceName)

		devices = append(devices, &DeviceInfo{
			ID:       deviceName,
			Type:     deviceType,
			Health:   health,
			Topology: topology,
		})
	}

	return devices, nil
}

// ============================================================
// 设备类型判断
// ============================================================

// getDeviceType 判断设备类型（IB或RoCE）
func getDeviceType(deviceName string) string {
	// ============================================================
	// 检查链路层类型
	// ============================================================
	linkLayerPath := fmt.Sprintf("/sys/class/infiniband/%s/ports/1/link_layer", deviceName)

	content, err := os.ReadFile(linkLayerPath)
	if err != nil {
		// 默认返回roce
		return "roce"
	}

	linkLayer := strings.TrimSpace(string(content))
	switch linkLayer {
	case "InfiniBand":
		return "ib"
	case "Ethernet":
		return "roce"
	default:
		return "roce"
	}
}

// ============================================================
// 拓扑信息获取
// ============================================================

// getDeviceTopology 获取设备NUMA拓扑
func getDeviceTopology(deviceName string) *TopologyInfo {
	// ============================================================
	// 读取NUMA节点
	// ============================================================
	numaPath := fmt.Sprintf("/sys/class/infiniband/%s/device/numa_node", deviceName)

	numaNode := 0
	content, err := os.ReadFile(numaPath)
	if err == nil {
		n, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err == nil && n >= 0 {
			numaNode = n
		}
	}

	// ============================================================
	// 读取PCI地址
	// ============================================================
	pciPath := fmt.Sprintf("/sys/class/infiniband/%s/device", deviceName)

	pciAddress := ""
	pciLink, err := os.Readlink(pciPath)
	if err == nil {
		pciAddress = filepath.Base(pciLink)
	}

	return &TopologyInfo{
		NUMANode:  numaNode,
		PCIAddress: pciAddress,
	}
}

// ============================================================
// 健康状态检查
// ============================================================

// CheckDeviceHealth 检查设备健康状态
func CheckDeviceHealth(deviceName string) bool {
	// ============================================================
	// 检查设备端口状态
	// ============================================================
	statePath := fmt.Sprintf("/sys/class/infiniband/%s/ports/1/state", deviceName)

	content, err := os.ReadFile(statePath)
	if err != nil {
		return false
	}

	state := strings.TrimSpace(string(content))
	// IB端口状态: "ACTIVE:4" 表示活跃
	// Mock实现：简单检查包含ACTIVE
	return strings.Contains(state, "ACTIVE")
}

// ============================================================
// Mock函数
// ============================================================

// MockDiscoverDevices 返回模拟设备列表（测试用）
func MockDiscoverDevices() []*DeviceInfo {
	return []*DeviceInfo{
		{
			ID:     "mlx5_0",
			Type:   "ib",
			Health: true,
			Topology: &TopologyInfo{
				NUMANode:  0,
				PCIAddress: "0000:05:00.0",
			},
		},
		{
			ID:     "mlx5_1",
			Type:   "roce",
			Health: true,
			Topology: &TopologyInfo{
				NUMANode:  1,
				PCIAddress: "0000:06:00.0",
			},
		},
	}
}

// MockGetDeviceList 返回模拟设备列表（命名符合规范）
func MockGetDeviceList() map[string]*DeviceInfo {
	devices := make(map[string]*DeviceInfo)
	for _, dev := range MockDiscoverDevices() {
		devices[dev.ID] = dev
	}
	return devices
}