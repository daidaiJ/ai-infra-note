// Package deviceuuid 实现设备 UUID 匹配调度插件
// 核心功能：允许 Pod 指定特定 GPU 设备 UUID，实现精确的设备绑定
package deviceuuid

import (
	"context"
	"strings"

	"github.com/example/gpu-scheduler/pkg/gpuinfo"
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// ============================================================
// 插件常量配置
// ============================================================
const (
	Name = "DeviceUUIDMatcher"

	// Pod 注解 Key：用于指定目标设备 UUID
	// 使用示例:
	//   annotations:
	//     scheduler.example.com/target-gpu-uuid: "GPU-abc123-0001"
	AnnotationTargetGPUUUID = "scheduler.example.com/target-gpu-uuid"

	// 支持多个设备 UUID（逗号分隔）
	//   annotations:
	//     scheduler.example.com/target-gpu-uuids: "GPU-abc123-0001,GPU-abc123-0002"
	AnnotationTargetGPUUUIDs = "scheduler.example.com/target-gpu-uuids"
)

// ============================================================
// 插件结构体
// ============================================================

// DeviceUUIDPlugin 将 Pod 调度到指定 UUID 设备所在的节点
// 实现接口: framework.FilterPlugin + framework.ScorePlugin
type DeviceUUIDPlugin struct {
	handle framework.Handle
}

// ============================================================
// 实现 framework.Plugin 接口
// ============================================================

func (p *DeviceUUIDPlugin) Name() string {
	return Name
}

// ============================================================
// 实现 framework.FilterPlugin 接口
// 核心逻辑：只允许 Pod 调度到目标设备所在的节点
// ============================================================

func (p *DeviceUUIDPlugin) Filter(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
) *framework.Status {

	nodeName := nodeInfo.Node().Name

	// === 步骤1: 检查 Pod 是否指定了目标设备 UUID ===
	targetUUIDs := p.getPodTargetUUIDs(pod)
	if len(targetUUIDs) == 0 {
		// Pod 未指定目标设备，交由其他插件处理
		return framework.NewStatus(framework.Success, "")
	}

	// === 步骤2: 检查节点是否包含目标设备 ===
	for _, uuid := range targetUUIDs {
		deviceNode := gpuinfo.GetNodeByDeviceUUID(uuid)

		// 设备不存在
		if deviceNode == "" {
			return framework.NewStatus(framework.Unschedulable,
				"指定的 GPU 设备不存在: "+uuid)
		}

		// 设备不在当前节点
		if deviceNode != nodeName {
			continue // 继续检查其他设备
		}

		// 找到匹配，节点通过
		return framework.NewStatus(framework.Success, "")
	}

	// 所有目标设备都不在此节点
	return framework.NewStatus(framework.Unschedulable,
		"节点不包含指定的 GPU 设备")
}

// ============================================================
// 实现 framework.ScorePlugin 接口
// 核心逻辑：目标设备所在节点得高分
// ============================================================

func (p *DeviceUUIDPlugin) Score(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeName string,
) (int64, *framework.Status) {

	targetUUIDs := p.getPodTargetUUIDs(pod)
	if len(targetUUIDs) == 0 {
		// 未指定目标设备，中性评分
		return 50, framework.NewStatus(framework.Success, "")
	}

	// 检查节点是否包含目标设备
	for _, uuid := range targetUUIDs {
		deviceNode := gpuinfo.GetNodeByDeviceUUID(uuid)
		if deviceNode == nodeName {
			// 目标设备在此节点，给予最高分
			return 100, framework.NewStatus(framework.Success, "")
		}
	}

	// 目标设备不在此节点，低分
	return 0, framework.NewStatus(framework.Success, "")
}

func (p *DeviceUUIDPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// ============================================================
// 辅助方法
// ============================================================

// getPodTargetUUIDs 从 Pod 注解中提取目标设备 UUID 列表
func (p *DeviceUUIDPlugin) getPodTargetUUIDs(pod *v1.Pod) []string {
	var uuids []string

	// 支持单设备注解
	if uuid, ok := pod.Annotations[AnnotationTargetGPUUUID]; ok && uuid != "" {
		uuids = append(uuids, uuid)
	}

	// 支持多设备注解（逗号分隔）
	if uuidsStr, ok := pod.Annotations[AnnotationTargetGPUUUIDs]; ok && uuidsStr != "" {
		for _, uuid := range strings.Split(uuidsStr, ",") {
			uuid = strings.TrimSpace(uuid)
			if uuid != "" {
				uuids = append(uuids, uuid)
			}
		}
	}

	return uuids
}

// ============================================================
// 插件工厂函数
// ============================================================

func New(_ context.Context, _ framework.ObjectToConfig, handle framework.Handle) (framework.Plugin, error) {
	return &DeviceUUIDPlugin{handle: handle}, nil
}