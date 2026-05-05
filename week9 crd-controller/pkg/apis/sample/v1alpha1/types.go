// Package v1alpha1 提供 SampleResource 自定义资源类型定义
//
// 本包定义了 SampleResource CRD 的 Go 类型，包括：
// - SampleResourceSpec: 期望状态（用户配置）
// - SampleResourceStatus: 实际状态（控制器上报）
// - SampleResource: 完整的 CR 对象
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ============================================================
// 常量定义
// ============================================================

const (
	// SampleResourceFinalizer Finalizer 名称
	// 用于拦截删除操作，确保外部资源被正确清理
	SampleResourceFinalizer = "sample.ai-infra/cleanup"
	
	// ConditionTypeReady 就绪条件类型
	ConditionTypeReady = "Ready"
	
	// ConditionTypeProgressing 进行中条件类型
	ConditionTypeProgressing = "Progressing"
)

// ============================================================
// SampleResourceSpec 定义用户期望状态
// ============================================================
// 【核心字段】
// - Replicas: 期望副本数
// - Feature: 启用的特性
// - Image: 容器镜像
//
// 【调用时机】
// 用户创建或更新 CR 时写入

type SampleResourceSpec struct {
	// ============================================================
	// 必需字段：核心配置，用户必须提供
	// ============================================================
	
	// Replicas 期望副本数
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Replicas int32 `json:"replicas"`
	
	// Feature 启用的特性名称
	// 可选值：advanced-scheduling, basic, none
	// +kubebuilder:validation:Enum=advanced-scheduling;basic;none
	Feature string `json:"feature"`
	
	// ============================================================
	// 可选字段：有默认值，用户可覆盖
	// ============================================================
	
	// Image 使用的容器镜像
	// +optional
	// +kubebuilder:default="nginx:latest"
	Image string `json:"image,omitempty"`
	
	// Port 服务暴露的端口
	// +optional
	// +kubebuilder:default=80
	Port int32 `json:"port,omitempty"`
}

// ============================================================
// SampleResourceStatus 定义实际状态
// ============================================================
// 【核心字段】
// - Phase: 当前阶段
// - Replicas: 实际副本数
// - Conditions: 详细条件
// - ObservedGeneration: 最后处理的 Generation
//
// 【调用时机】
// 控制器 Reconcile 成功后写入

type SampleResourceStatus struct {
	// Phase 当前阶段
	// 可能值：Pending, Creating, Running, Failed, Terminating
	Phase string `json:"phase,omitempty"`
	
	// Replicas 实际副本数
	Replicas int32 `json:"replicas,omitempty"`
	
	// AvailableReplicas 可用副本数
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
	
	// ObservedGeneration 最后处理的 Generation
	// 用于判断 Status 是否对应最新 Spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	
	// Conditions 详细条件
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	
	// LastTransitionTime 最后状态转换时间
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// ============================================================
// SampleResource 完整的 CR 对象
// ============================================================
// 【使用场景】
// 用户通过 YAML 创建，控制器通过 Go 类型读取
//
// 【生命周期】
// 创建 → 调谐 → 运行 → 更新 → 删除

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=sr
type SampleResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec   SampleResourceSpec   `json:"spec,omitempty"`
	Status SampleResourceStatus `json:"status,omitempty"`
}

// ============================================================
// SampleResourceList CR 列表
// ============================================================

// +kubebuilder:object:root=true
type SampleResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SampleResource `json:"items"`
}

// ============================================================
// init 注册类型到 Scheme
// ============================================================

func init() {
	// 此函数将在 AddToScheme 时自动调用
}
