// Package v1alpha1 提供 SampleResource 注册逻辑
//
// 本文件负责将 v1alpha1 组的类型注册到 Kubernetes Scheme 中，
// 使得 controller-runtime 能够序列化和反序列化自定义资源。

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// ============================================================
// 常量定义
// ============================================================

const (
	// GroupVersion 组版本
	// 格式：group/version
	GroupVersion = "sample.ai-infra/v1alpha1"
	
	// Group 组名称
	Group = "sample.ai-infra"
)

// ============================================================
// 变量声明
// ============================================================

var (
	// GroupVersionVariable 组版本对象
	// 用于反射和类型注册
	GroupVersionVariable = schema.FromAPIVersionAndKind(Group, "SampleResource")
	
	// AddToScheme 将类型添加到 Scheme 的函数
	// 在 main.go 的 init() 中调用
	AddToScheme = scheme.Builder{GroupVersion: GroupVersion}.AddToScheme
	
	// SchemeBuilder 用于构建 Scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
)

func init() {
	// 注册 SampleResource 和 SampleResourceList
	SchemeBuilder.Register(&SampleResource{}, &SampleResourceList{})
}
