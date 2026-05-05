// Package gpuutilization 实现 GPU 利用率感知调度插件
// 核心功能：优先将 GPU 任务调度到平均利用率较低的节点，实现负载均衡
package gpuutilization

import (
	"context"

	"github.com/example/gpu-scheduler/pkg/gpuinfo"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// ============================================================
// 插件名称常量
// ============================================================
const (
	Name = "GPUUtilizationBalancer"
)

// ============================================================
// 插件结构体
// ============================================================

// GPUUtilizationPlugin 优先调度到 GPU 利用率低的节点
// 实现接口: framework.FilterPlugin + framework.ScorePlugin
type GPUUtilizationPlugin struct {
	// handle 提供访问调度器内部资源的能力
	// 如 SharedLister 获取节点列表、ClientSet 访问 K8s API 等
	handle framework.Handle
}

// ============================================================
// 必须实现 framework.Plugin 接口
// ============================================================

// Name 返回插件名称，用于配置文件中引用
func (p *GPUUtilizationPlugin) Name() string {
	return Name
}

// ============================================================
// 实现 framework.FilterPlugin 接口
// 作用：过滤掉不满足条件的节点（预选阶段）
// ============================================================

// Filter 在预选阶段执行，过滤不可用节点
// 返回 Unschedulable 表示节点被过滤，Success 表示通过
func (p *GPUUtilizationPlugin) Filter(
	ctx context.Context,
	state *framework.CycleState, // 调度周期状态，可在同一调度周期的不同插件间共享数据
	pod *v1.Pod,                 // 待调度的 Pod
	nodeInfo *framework.NodeInfo, // 候选节点信息
) *framework.Status {

	nodeName := nodeInfo.Node().Name

	// === 步骤1: 检查节点是否有 GPU 资源 ===
	if !p.hasGPUResources(nodeInfo) {
		// 没有 GPU 资源的节点直接通过，交由其他插件处理
		return framework.NewStatus(framework.Success, "")
	}

	// === 步骤2: 检查 GPU 设备健康状态 ===
	// 可从设备监控服务获取健康信息，过滤不健康设备
	if !p.isGPUHealthy(nodeName) {
		return framework.NewStatus(framework.Unschedulable, "节点 GPU 设备不健康")
	}

	// === 步骤3: 检查显存是否满足需求 ===
	// Pod 可能声明了显存需求（通过扩展资源或注解）
	requestedMemory := p.getPodGPUMemoryRequest(pod)
	if requestedMemory > 0 {
		availableMemory := p.getAvailableGPUMemory(nodeName)
		if availableMemory < requestedMemory {
			return framework.NewStatus(framework.Unschedulable, "节点显存不足")
		}
	}

	return framework.NewStatus(framework.Success, "")
}

// ============================================================
// 实现 framework.ScorePlugin 接口
// 作用：为通过 Filter 的节点打分（优选阶段），分数越高越优先
// ============================================================

// Score 为节点打分，分数范围 [0, 100]
// 利用率越低，分数越高，实现负载均衡
func (p *GPUUtilizationPlugin) Score(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeName string,
) (int64, *framework.Status) {

	// === 核心逻辑: 利用率低 → 分数高 ===
	// 从监控系统获取节点的 GPU 平均利用率
	utilization := gpuinfo.GetNodeGPUUtilization(nodeName)

	// 公式: Score = 100 - utilization
	// 利用率 0%  → Score = 100 (最高)
	// 利用率 100% → Score = 0   (最低)
	score := int64(100 - utilization)

	return score, framework.NewStatus(framework.Success, "")
}

// ScoreExtensions 返回扩展接口，用于分数归一化
// 返回 nil 表示不需要归一化（我们的分数已经在 0-100 范围内）
func (p *GPUUtilizationPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// ============================================================
// 辅助方法（示意性实现）
// ============================================================

// hasGPUResources 检查节点是否有 GPU 资源
func (p *GPUUtilizationPlugin) hasGPUResources(nodeInfo *framework.NodeInfo) bool {
	node := nodeInfo.Node()
	if node == nil {
		return false
	}
	// 检查节点是否有 nvidia.com/gpu 资源
	if gpu, ok := node.Status.Capacity[v1.ResourceName("nvidia.com/gpu")]; ok {
		return !gpu.IsZero()
	}
	return false
}

// isGPUHealthy 检查节点 GPU 设备健康状态
func (p *GPUUtilizationPlugin) isGPUHealthy(nodeName string) bool {
	// 从设备监控服务或 Prometheus 查询健康状态
	// Mock: 假设所有设备都健康
	return true
}

// getAvailableGPUMemory 获取节点可用显存
func (p *GPUUtilizationPlugin) getAvailableGPUMemory(nodeName string) int64 {
	// 从监控系统获取可用显存
	// Mock: 返回模拟值
	return 65536 // MiB
}

// getPodGPUMemoryRequest 获取 Pod 的显存需求
func (p *GPUUtilizationPlugin) getPodGPUMemoryRequest(pod *v1.Pod) int64 {
	// 可从 Pod 注解或扩展资源获取
	// 例如: pod.annotations["gpu-memory-request"]
	return 0
}

// ============================================================
// 插件工厂函数
// 由调度器框架调用，创建插件实例
// ============================================================

func New(_ context.Context, _ framework.ObjectToConfig, handle framework.Handle) (framework.Plugin, error) {
	return &GPUUtilizationPlugin{handle: handle}, nil
}