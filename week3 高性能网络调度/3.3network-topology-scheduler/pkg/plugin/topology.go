// Package plugin 网络拓扑感知调度插件
package plugin

import (
	"context"

	"network-topology-scheduler/pkg/numa"
	"network-topology-scheduler/pkg/rack"
)

// ============================================================
// 常量定义
// ============================================================

const (
	// PluginName 插件名称
	PluginName = "NetworkTopology"

	// FilterName Filter插件名称
	FilterName = "NetworkTopologyFilter"

	// ScoreName Score插件名称
	ScoreName = "NetworkTopologyScore"

	// MaxScore 最高分数
	MaxScore = 100
)

// ============================================================
// Mock接口定义（示意性）
// ============================================================

// FilterResult Filter结果
type FilterResult struct {
	Result  int
	Reasons []string
}

// CycleState 调度周期状态
type CycleState struct{}

// NodeInfo 节点信息
type NodeInfo struct {
	Node *NodeSpec
}

// NodeSpec 节点规格
type NodeSpec struct {
	Name string
}

// Pod Pod规格
type Pod struct {
	Spec PodSpec
}

// PodSpec Pod规格详情
type PodSpec struct {
	Containers []ContainerSpec
}

// ContainerSpec 容器规格
type ContainerSpec struct {
	Resources ResourceRequirements
}

// ResourceRequirements 资源需求
type ResourceRequirements struct {
	Limits   map[string]int64
	Requests map[string]int64
}

// ============================================================
// Filter结果常量
// ============================================================

const (
	FilterSuccess     = 0
	FilterUnsuitable  = 1
)

// ============================================================
// 结构体定义
// ============================================================

// NetworkTopologyFilter 网络拓扑过滤插件
type NetworkTopologyFilter struct {
	// numaFilter NUMA亲和性过滤器
	numaFilter *numa.NUMAFilter

	// topologyMap 节点拓扑信息
	topologyMap map[string]*rack.NodeTopologyInfo
}

// NetworkTopologyScore 网络拓扑打分插件
type NetworkTopologyScore struct {
	// topologyMap 节点拓扑信息
	topologyMap map[string]*rack.NodeTopologyInfo

	// numaMap NUMA拓扑信息
	numaMap map[string]*numa.NodeNUMAInfo
}

// ============================================================
// Filter接口实现
// ============================================================

// NewNetworkTopologyFilter 创建拓扑过滤插件
func NewNetworkTopologyFilter() *NetworkTopologyFilter {
	return &NetworkTopologyFilter{
		numaFilter:   numa.NewNUMAFilter(),
		topologyMap:  rack.MockGetTopologyMap(),
	}
}

// Filter 综合拓扑过滤
// 【调用时机】调度器过滤阶段，排除不符合拓扑要求的节点
func (p *NetworkTopologyFilter) Filter(ctx context.Context, state *CycleState, pod *Pod, nodeInfo *NodeInfo) (*FilterResult, error) {
	// === 步骤1: NUMA亲和性检查 ===
	numaResult := p.numaFilter.CheckNUMAAffinity(pod, nodeInfo.Node.Name)
	if !numaResult.Success {
		return &FilterResult{
			Result:  FilterUnsuitable,
			Reasons: numaResult.Reasons,
		}, nil
	}

	// === 步骤2: 机架可用性检查（多节点Pod） ===
	if needsMultiNode(pod) {
		rackResult := p.checkRackAvailability(pod, nodeInfo.Node.Name)
		if !rackResult.Success {
			return &FilterResult{
				Result:  FilterUnsuitable,
				Reasons: rackResult.Reasons,
			}, nil
		}
	}

	return &FilterResult{Result: FilterSuccess}, nil
}

// checkRackAvailability 检查机架内是否有足够节点
func (p *NetworkTopologyFilter) checkRackAvailability(pod *Pod, nodeName string) *FilterResult {
	// ============================================================
	// 对于分布式训练Pod，检查同机架是否有足够节点
	// ============================================================

	nodeTopology := p.topologyMap[nodeName]

	// 计算同机架可用节点数
	sameRackNodes := 0
	for _, topology := range p.topologyMap {
		if topology.RackID == nodeTopology.RackID {
			sameRackNodes++
		}
	}

	// Mock实现：假设需要3个节点
	requiredNodes := 3

	if sameRackNodes < requiredNodes {
		return &FilterResult{
			Result:  FilterUnsuitable,
			Reasons: []string{"Insufficient nodes in same rack"},
		}
	}

	return &FilterResult{Result: FilterSuccess}
}

// ============================================================
// Score接口实现
// ============================================================

// NewNetworkTopologyScore 创建拓扑打分插件
func NewNetworkTopologyScore() *NetworkTopologyScore {
	return &NetworkTopologyScore{
		topologyMap: rack.MockGetTopologyMap(),
		numaMap:     numa.MockGetNUMAMap(),
	}
}

// Score 根据拓扑距离打分
// 【调用时机】调度器打分阶段，计算节点的拓扑分数
func (p *NetworkTopologyScore) Score(ctx context.Context, state *CycleState, pod *Pod, nodeInfo *NodeInfo) (int64, error) {
	// ============================================================
	// 打分规则（满分100）
	// ============================================================

	// === 步骤1: NUMA亲和性打分（权重60%） ===
	numaScore := p.calculateNUMAScore(pod, nodeInfo.Node.Name)

	// === 步骤2: 机架距离打分（权重40%） ===
	rackScore := p.calculateRackScore(pod, nodeInfo.Node.Name)

	// === 步骤3: 综合分数 ===
	totalScore := numaScore*60/100 + rackScore*40/100

	return totalScore, nil
}

// calculateNUMAScore 计算NUMA亲和分数
func (p *NetworkTopologyScore) calculateNUMAScore(pod *Pod, nodeName string) int64 {
	// ============================================================
	// NUMA打分规则（满分100）
	// ============================================================
	// - RDMA设备与CPU同NUMA: 100分
	// - RDMA设备与CPU跨NUMA: 50分
	// ============================================================

	cpuRequest := getCPURequest(pod)
	rdmaRequest := getRDMARequest(pod)

	if rdmaRequest == 0 || cpuRequest == 0 {
		return MaxScore // 不涉及NUMA约束
	}

	// Mock实现：假设完美NUMA亲和
	// 生产环境：检查实际CPU NUMA分布与RDMA NUMA匹配
	return MaxScore
}

// calculateRackScore 计算机架距离分数
func (p *NetworkTopologyScore) calculateRackScore(pod *Pod, nodeName string) int64 {
	// ============================================================
	// 机架打分规则（满分100）
	// ============================================================
	// - 同机架: 100分
	// - 跨机架（同核心交换机）: 70分
	// - 跨核心交换机: 50分
	// ============================================================

	if !needsMultiNode(pod) {
		return MaxScore // 单节点Pod，机架不影响
	}

	// Mock实现：假设同机架
	return MaxScore
}

// ============================================================
// 辅助函数
// ============================================================

// needsMultiNode 判断是否需要多节点
func needsMultiNode(pod *Pod) bool {
	// Mock实现：示意性判断
	// 生产环境：根据Pod annotation或Job配置判断
	return false
}

// getCPURequest 获取CPU请求
func getCPURequest(pod *Pod) int64 {
	// Mock实现：示意性获取
	// 生产环境：解析Pod spec的resources
	return 8
}

// getRDMARequest 获取RDMA请求
func getRDMARequest(pod *Pod) int64 {
	// Mock实现：示意性获取
	return 1
}

// ============================================================
// Mock结构体
// ============================================================

// CheckResult 检查结果
type CheckResult struct {
	Success bool
	Reasons []string
}