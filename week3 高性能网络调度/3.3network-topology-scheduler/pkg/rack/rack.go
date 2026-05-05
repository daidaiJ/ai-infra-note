// Package rack 机架感知算法
package rack

// ============================================================
// 结构体定义
// ============================================================

// RackTopology 机架拓扑信息
type RackTopology struct {
	// RackID 机架ID
	RackID string

	// SwitchID 机架交换机ID
	SwitchID string

	// Nodes 机架内节点列表
	Nodes []string
}

// NodeTopologyInfo 节点拓扑信息
type NodeTopologyInfo struct {
	// NodeName 节点名称
	NodeName string

	// RackID 所属机架ID
	RackID string

	// SwitchID 所属交换机ID
	SwitchID string

	// CoreSwitchID 核心交换机ID
	CoreSwitchID string

	// HopCountToOther 到其他节点的跳数映射
	HopCountToOther map[string]int // nodeName -> hopCount
}

// ============================================================
// 拓扑距离计算
// ============================================================

// CalculateHopCount 计算两个节点间的网络跳数
func CalculateHopCount(node1, node2 *NodeTopologyInfo) int {
	// ============================================================
	// 跳数计算规则
	// ============================================================
	// - 同节点:      0跳
	// - 同机架:      1跳（通过机架交换机）
	// - 同核心交换机: 2跳（机架交换机→核心→机架交换机）
	// - 跨核心交换机: 3跳+
	// ============================================================

	if node1.NodeName == node2.NodeName {
		return 0 // 同节点
	}

	if node1.RackID == node2.RackID {
		return 1 // 同机架
	}

	if node1.CoreSwitchID == node2.CoreSwitchID {
		return 2 // 同核心交换机，不同机架
	}

	return 3 // 跨核心交换机
}

// CalculateRackDistance 计算机架距离
func CalculateRackDistance(node1, node2 *NodeTopologyInfo) int {
	// ============================================================
	// 机架距离计算
	// ============================================================
	if node1.RackID == node2.RackID {
		return 0 // 同机架
	}

	// Mock实现：假设相邻机架距离为1
	// 生产环境：可根据数据中心实际拓扑计算
	return 1
}

// CalculateAverageHopCount 计算到一组节点的平均跳数
func CalculateAverageHopCount(sourceNode *NodeTopologyInfo, targetNodes []string, topologyMap map[string]*NodeTopologyInfo) float64 {
	// ============================================================
	// 平均跳数计算（用于多节点任务打分）
	// ============================================================
	if len(targetNodes) == 0 {
		return 0
	}

	totalHops := 0
	for _, targetNodeName := range targetNodes {
		targetNode := topologyMap[targetNodeName]
		if targetNode != nil {
			hops := CalculateHopCount(sourceNode, targetNode)
			totalHops += hops
		}
	}

	return float64(totalHops) / float64(len(targetNodes))
}

// ============================================================
// 机架内节点查找
// ============================================================

// FindNodesInSameRack 查找同机架的节点
func FindNodesInSameRack(nodeName string, topologyMap map[string]*NodeTopologyInfo) []string {
	// ============================================================
	// 查找同机架节点（用于多节点任务调度）
	// ============================================================
	nodeTopology := topologyMap[nodeName]
	if nodeTopology == nil {
		return []string{}
	}

	sameRackNodes := make([]string, 0)
	for name, topology := range topologyMap {
		if topology.RackID == nodeTopology.RackID {
			sameRackNodes = append(sameRackNodes, name)
		}
	}

	return sameRackNodes
}

// FindBestRackForMultiNode 为多节点任务选择最佳机架
func FindBestRackForMultiNode(requiredNodes int, topologyMap map[string]*NodeTopologyInfo) string {
	// ============================================================
	// 选择有足够节点的机架
	// ============================================================
	rackNodeCount := make(map[string]int)

	for _, topology := range topologyMap {
		rackNodeCount[topology.RackID]++
	}

	for rackID, count := range rackNodeCount {
		if count >= requiredNodes {
			return rackID
		}
	}

	return "" // 无足够节点的机架
}

// ============================================================
// Mock函数
// ============================================================

// MockGetTopologyMap 返回模拟拓扑信息（测试用）
func MockGetTopologyMap() map[string]*NodeTopologyInfo {
	return map[string]*NodeTopologyInfo{
		"node-1": {
			NodeName:       "node-1",
			RackID:         "rack-1",
			SwitchID:       "switch-1",
			CoreSwitchID:   "core-switch-1",
			HopCountToOther: map[string]int{
				"node-1": 0,
				"node-2": 1,
				"node-3": 1,
				"node-4": 2,
				"node-5": 2,
			},
		},
		"node-2": {
			NodeName:       "node-2",
			RackID:         "rack-1",
			SwitchID:       "switch-1",
			CoreSwitchID:   "core-switch-1",
			HopCountToOther: map[string]int{
				"node-1": 1,
				"node-2": 0,
				"node-3": 1,
				"node-4": 2,
				"node-5": 2,
			},
		},
		"node-3": {
			NodeName:       "node-3",
			RackID:         "rack-1",
			SwitchID:       "switch-1",
			CoreSwitchID:   "core-switch-1",
			HopCountToOther: map[string]int{
				"node-1": 1,
				"node-2": 1,
				"node-3": 0,
				"node-4": 2,
				"node-5": 2,
			},
		},
		"node-4": {
			NodeName:       "node-4",
			RackID:         "rack-2",
			SwitchID:       "switch-2",
			CoreSwitchID:   "core-switch-1",
			HopCountToOther: map[string]int{
				"node-1": 2,
				"node-2": 2,
				"node-3": 2,
				"node-4": 0,
				"node-5": 1,
			},
		},
		"node-5": {
			NodeName:       "node-5",
			RackID:         "rack-2",
			SwitchID:       "switch-2",
			CoreSwitchID:   "core-switch-1",
			HopCountToOther: map[string]int{
				"node-1": 2,
				"node-2": 2,
				"node-3": 2,
				"node-4": 1,
				"node-5": 0,
			},
		},
	}
}

// MockGetRackTopology 返回机架拓扑
func MockGetRackTopology() map[string]*RackTopology {
	return map[string]*RackTopology{
		"rack-1": {
			RackID:   "rack-1",
			SwitchID: "switch-1",
			Nodes:    []string{"node-1", "node-2", "node-3"},
		},
		"rack-2": {
			RackID:   "rack-2",
			SwitchID: "switch-2",
			Nodes:    []string{"node-4", "node-5"},
		},
	}
}