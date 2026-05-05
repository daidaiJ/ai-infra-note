// Package numa NUMA亲和性算法
package numa

// ============================================================
// 结构体定义
// ============================================================

// NUMATopology NUMA节点拓扑信息
type NUMATopology struct {
	// NodeID NUMA节点编号
	NodeID int

	// CPUs 该NUMA节点的CPU列表
	CPUs []int

	// Devices 该NUMA节点的设备列表（GPU、网卡）
	Devices []DeviceInfo
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	ID      string
	NUMANode int
}

// NodeNUMAInfo 节点NUMA拓扑信息
type NodeNUMAInfo struct {
	// NUMANodes NUMA节点列表
	NUMANodes []NUMATopology

	// DeviceNUMAMap 设备到NUMA的映射
	DeviceNUMAMap map[string]int // deviceID -> numaNode
}

// NUMAFilter NUMA亲和性过滤器
type NUMAFilter struct {
	// topologyMap 节点NUMA拓扑信息
	topologyMap map[string]*NodeNUMAInfo
}

// ============================================================
// NUMA亲和性检查
// ============================================================

// NewNUMAFilter 创建NUMA过滤器
func NewNUMAFilter() *NUMAFilter {
	return &NUMAFilter{
		topologyMap: MockGetNUMAMap(),
	}
}

// CheckNUMAAffinity 检查NUMA亲和性
// 【核心逻辑】确保RDMA设备与请求的CPU在同一NUMA节点
func (f *NUMAFilter) CheckNUMAAffinity(pod *Pod, nodeName string) *CheckResult {
	// ============================================================
	// NUMA亲和性检查流程
	// ============================================================

	// === 步骤1: 获取Pod资源请求 ===
	cpuRequest := getCPURequest(pod)
	rdmaRequest := getRDMARequest(pod)

	if rdmaRequest == 0 {
		// 不请求RDMA，跳过NUMA检查
		return &CheckResult{Success: true}
	}

	// === 步骤2: 获取节点NUMA拓扑 ===
	nodeTopology := f.topologyMap[nodeName]

	// === 步骤3: 查找RDMA设备的NUMA节点 ===
	rdmaNUMA := nodeTopology.DeviceNUMAMap["mlx5_0"]

	// === 步骤4: 检查NUMA节点的CPU资源 ===
	for _, numaNode := range nodeTopology.NUMANodes {
		if numaNode.NodeID == rdmaNUMA {
			numaCPUCount := len(numaNode.CPUs)

			// === 步骤5: 判断CPU是否足够 ===
			if numaCPUCount >= cpuRequest {
				return &CheckResult{Success: true}
			}
		}
	}

	return &CheckResult{
		Success: false,
		Reasons: []string{"Insufficient CPUs on RDMA NUMA node"},
	}
}

// ============================================================
// Mock函数
// ============================================================

// MockGetNUMAMap 返回模拟NUMA拓扑（测试用）
func MockGetNUMAMap() map[string]*NodeNUMAInfo {
	return map[string]*NodeNUMAInfo{
		"node-1": {
			NUMANodes: []NUMATopology{
				{
					NodeID: 0,
					CPUs:   []int{0, 1, 2, 3, 4, 5, 6, 7},
					Devices: []DeviceInfo{
						{ID: "mlx5_0", NUMANode: 0},
					},
				},
				{
					NodeID: 1,
					CPUs:   []int{8, 9, 10, 11, 12, 13, 14, 15},
					Devices: []DeviceInfo{
						{ID: "mlx5_1", NUMANode: 1},
					},
				},
			},
			DeviceNUMAMap: map[string]int{
				"mlx5_0": 0,
				"mlx5_1": 1,
			},
		},
		"node-2": {
			NUMANodes: []NUMATopology{
				{
					NodeID: 0,
					CPUs:   []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
					Devices: []DeviceInfo{
						{ID: "mlx5_0", NUMANode: 0},
					},
				},
			},
			DeviceNUMAMap: map[string]int{
				"mlx5_0": 0,
			},
		},
	}
}

// Mock接口定义（示意性）
type Pod struct{}
type CheckResult struct {
	Success bool
	Reasons []string
}

func getCPURequest(pod *Pod) int64 { return 8 }
func getRDMARequest(pod *Pod) int64 { return 1 }