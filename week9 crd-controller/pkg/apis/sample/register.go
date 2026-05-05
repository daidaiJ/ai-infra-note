// Package sample 提供 SampleResource API 组注册
//
// 本文件是组级别的注册入口，v1alpha1 子包负责具体类型定义。

package sample

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	
	"example.com/week9-crd-controller/pkg/apis/sample/v1alpha1"
)

// ============================================================
// 常量定义
// ============================================================

const (
	// GroupName SampleResource 组名称
	GroupName = "sample.ai-infra"
)

// ============================================================
// 变量声明
// ============================================================

var (
	// SchemeGroupVersion 组版本对象
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}
	
	// AddToScheme 将类型添加到 Scheme 的函数
	AddToScheme = v1alpha1.AddToScheme
)
