// Package controller 提供 SampleResource 控制器实现
//
// 本包实现了 SampleResource CRD 的控制器，负责：
// - 监听 CR 的创建、更新、删除 事件
// - 根据 Spec 创建和管理子资源（Deployment、Service）
// - 更新 Status 反映实际状态
// - 上报 Event 记录关键操作
package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	samplev1alpha1 "example.com/week9-crd-controller/pkg/apis/sample/v1alpha1"
)

// ============================================================
// 常量定义
// ============================================================

const (
	// finalizerName Finalizer 名称
	// 用于拦截删除操作，确保外部资源被正确清理
	finalizerName = samplev1alpha1.SampleResourceFinalizer
	
	// controllerName 控制器名称
	// 用于日志和事件标识
	controllerName = "sample-controller"
)

// ============================================================
// SampleResourceReconciler 控制器结构体
// ============================================================
// 实现 ctrl.Reconciler 接口
// 【使用场景】
// 监听 SampleResource 变化，调谐实际状态与期望状态一致
//
// 【调用时机】
// - SampleResource 创建时
// - SampleResource 更新时
// - SampleResource 删除时（通过 Finalizer）
// - 定时重新调谐时

type SampleResourceReconciler struct {
	// Client Kubernetes API 客户端
	Client client.Client
	
	// Scheme 运行时类型注册表
	Scheme *runtime.Scheme
	
	// Recorder 事件记录器
	Recorder record.EventRecorder
}

// ============================================================
// Reconcile 调谐函数
// ============================================================
// 控制器的核心逻辑，负责将实际状态调谐到期望状态
//
// 【调用时机】
// - Informer 监听到 CR 变化
// - WorkQueue 出队
//
// 【返回类型】
// - Result{}, nil: 成功，不再入队
// - Result{Requeue: true}: 立即重新入队
// - Result{RequeueAfter: t}: 延迟 t 后入队
// - Result{}, err: 错误，指数退避入队
//
// 【幂等性】
// 本函数必须支持安全重复执行，多次执行结果相同

func (r *SampleResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 获取日志上下文
	log := log.FromContext(ctx)
	
	// === 步骤1: 获取 CR 实例 ===
	var sr samplev1alpha1.SampleResource
	if err := r.Client.Get(ctx, req.NamespacedName, &sr); err != nil {
		if apierrors.IsNotFound(err) {
			// CR 已被删除，不再重试
			return ctrl.Result{}, nil
		}
		// 其他错误（如网络问题），返回重试
		log.Error(err, "Failed to get SampleResource")
		return ctrl.Result{}, err
	}
	
	// === 步骤2: 检查删除标记（DeletionTimestamp） ===
	if !sr.DeletionTimestamp.IsZero() {
		// CR 被标记删除，执行清理逻辑
		log.Info("SampleResource is being deleted", "name", sr.Name)
		return r.handleDeletion(ctx, &sr)
	}
	
	// === 步骤3: 添加 Finalizer（如果尚未添加） ===
	if !controllerutil.ContainsFinalizer(&sr, finalizerName) {
		controllerutil.AddFinalizer(&sr, finalizerName)
		if err := r.Client.Update(ctx, &sr); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Added finalizer", "name", sr.Name)
	}
	
	// === 步骤4: 执行调谐逻辑 ===
	// 根据 Spec 创建/更新子资源
	result, err := r.reconcileResources(ctx, &sr)
	if err != nil {
		// 调谐失败，更新 Status 为 Failed
		r.updateStatus(ctx, &sr, "Failed", sr.Spec.Replicas, 0, err.Error())
		r.Recorder.Event(&sr, corev1.EventTypeWarning, "FailedReconcile", err.Error())
		return result, err
	}
	
	// === 步骤5: 更新 Status 为 Ready ===
	r.updateStatus(ctx, &sr, "Running", sr.Spec.Replicas, sr.Spec.Replicas, "ReconcileSuccess")
	r.Recorder.Event(&sr, corev1.EventTypeNormal, "Reconciled", "SampleResource reconciled successfully")
	
	return result, nil
}

// ============================================================
// reconcileResources 调谐子资源
// ============================================================
// 根据 Spec 创建/更新 Deployment 和 Service
//
// 【调用时机】
// Reconcile 函数中调用
//
// 【幂等性】
// 创建前先检查是否存在，避免重复创建

func (r *SampleResourceReconciler) reconcileResources(ctx context.Context, sr *samplev1alpha1.SampleResource) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	
	// === 步骤1: 创建/更新 Deployment ===
	if err := r.reconcileDeployment(ctx, sr); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{}, err
	}
	
	// === 步骤2: 创建/更新 Service ===
	if err := r.reconcileService(ctx, sr); err != nil {
		log.Error(err, "Failed to reconcile Service")
		return ctrl.Result{}, err
	}
	
	// === 步骤3: 返回成功 ===
	return ctrl.Result{}, nil
}

// ============================================================
// reconcileDeployment 调谐 Deployment
// ============================================================
// 创建或更新 Deployment，确保与 Spec 一致
//
// 【调用时机】
// reconcileResources 中调用

func (r *SampleResourceReconciler) reconcileDeployment(ctx context.Context, sr *samplev1alpha1.SampleResource) error {
	log := log.FromContext(ctx)
	
	// === 步骤1: 检查 Deployment 是否已存在 ===
	var existing appsv1.Deployment
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      sr.Name,
		Namespace: sr.Namespace,
	}, &existing)
	
	if err == nil {
		// Deployment 已存在，检查是否需要更新
		if existing.Spec.Replicas != nil && *existing.Spec.Replicas == sr.Spec.Replicas {
			log.Info("Deployment already up-to-date", "name", sr.Name)
			return nil
		}
		// 更新副本数
		existing.Spec.Replicas = &sr.Spec.Replicas
		if err := r.Client.Update(ctx, &existing); err != nil {
			if apierrors.IsConflict(err) {
				// 冲突：其他协程已更新，重试
				log.Info("Conflict detected, will retry", "name", sr.Name)
				return fmt.Errorf("conflict updating Deployment: %w", err)
			}
			return fmt.Errorf("failed to update Deployment: %w", err)
		}
		log.Info("Updated Deployment", "name", sr.Name, "replicas", sr.Spec.Replicas)
		r.Recorder.Eventf(sr, corev1.EventTypeNormal, "Updated", "Updated Deployment %s", sr.Name)
		return nil
	}
	
	if !apierrors.IsNotFound(err) {
		// 其他错误（非不存在）
		return fmt.Errorf("failed to get Deployment: %w", err)
	}
	
	// === 步骤2: Deployment 不存在，创建 ===
	deployment := r.buildDeployment(sr)
	
	// 设置 OwnerReference（级联删除）
	if err := controllerutil.SetControllerReference(sr, deployment, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}
	
	if err := r.Client.Create(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create Deployment: %w", err)
	}
	
	log.Info("Created Deployment", "name", sr.Name)
	r.Recorder.Eventf(sr, corev1.EventTypeNormal, "Created", "Created Deployment %s", sr.Name)
	return nil
}

// ============================================================
// reconcileService 调谐 Service
// ============================================================
// 创建或更新 Service，暴露 Deployment
//
// 【调用时机】
// reconcileResources 中调用

func (r *SampleResourceReconciler) reconcileService(ctx context.Context, sr *samplev1alpha1.SampleResource) error {
	log := log.FromContext(ctx)
	
	// === 步骤1: 检查 Service 是否已存在 ===
	var existing corev1.Service
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      sr.Name,
		Namespace: sr.Namespace,
	}, &existing)
	
	if err == nil {
		// Service 已存在
		log.Info("Service already exists", "name", sr.Name)
		return nil
	}
	
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get Service: %w", err)
	}
	
	// === 步骤2: Service 不存在，创建 ===
	service := r.buildService(sr)
	
	// 设置 OwnerReference
	if err := controllerutil.SetControllerReference(sr, service, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}
	
	if err := r.Client.Create(ctx, service); err != nil {
		return fmt.Errorf("failed to create Service: %w", err)
	}
	
	log.Info("Created Service", "name", sr.Name)
	r.Recorder.Eventf(sr, corev1.EventTypeNormal, "Created", "Created Service %s", sr.Name)
	return nil
}

// ============================================================
// handleDeletion 处理删除逻辑
// ============================================================
// 当 CR 被标记删除时，清理外部资源并移除 Finalizer
//
// 【调用时机】
// Reconcile 检测到 DeletionTimestamp 不为零
//
// 【流程】
// 1. 清理外部资源（Deployment/Service 等，由 OwnerReference 自动处理）
// 2. 移除 Finalizer
// 3. API Server 自动完成删除

func (r *SampleResourceReconciler) handleDeletion(ctx context.Context, sr *samplev1alpha1.SampleResource) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	
	// === 步骤1: 清理外部资源 ===
	// 由于使用了 OwnerReference，Deployment/Service 会自动删除
	// 如需清理其他外部资源（如外部 API），在此添加逻辑
	
	// === 步骤2: 移除 Finalizer ===
	if controllerutil.ContainsFinalizer(sr, finalizerName) {
		controllerutil.RemoveFinalizer(sr, finalizerName)
		if err := r.Client.Update(ctx, sr); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Removed finalizer", "name", sr.Name)
		r.Recorder.Eventf(sr, corev1.EventTypeNormal, "Deleted", "Deleted SampleResource %s", sr.Name)
	}
	
	// === 步骤3: 返回成功，API Server 将完成删除 ===
	return ctrl.Result{}, nil
}

// ============================================================
// updateStatus 更新 Status
// ============================================================
// 更新 SampleResource 的 Status 字段
//
// 【调用时机】
// Reconcile 成功后/失败后调用
//
// 【最佳实践】
// 使用 Status().Update() 而非 Update()，避免覆盖 Spec

func (r *SampleResourceReconciler) updateStatus(
	ctx context.Context,
	sr *samplev1alpha1.SampleResource,
	phase string,
	replicas int32,
	availableReplicas int32,
	message string,
) {
	log := log.FromContext(ctx)
	
	// === 步骤1: 构建 Status ===
	sr.Status.Phase = phase
	sr.Status.Replicas = replicas
	sr.Status.AvailableReplicas = availableReplicas
	sr.Status.ObservedGeneration = sr.Generation
	sr.Status.LastTransitionTime = metav1.Now()
	
	// === 步骤2: 更新 Conditions ===
	conditionType := samplev1alpha1.ConditionTypeReady
	conditionStatus := metav1.ConditionFalse
	reason := "ReconcileFailed"
	
	if phase == "Running" {
		conditionStatus = metav1.ConditionTrue
		reason = "ReconcileSuccess"
	}
	
	sr.Status.Conditions = []metav1.Condition{
		{
			Type:               conditionType,
			Status:             conditionStatus,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: sr.Status.LastTransitionTime,
			ObservedGeneration: sr.Generation,
		},
	}
	
	// === 步骤3: 更新 Status 子资源 ===
	if err := r.Client.Status().Update(ctx, sr); err != nil {
		if apierrors.IsConflict(err) {
			// 冲突：其他协程已更新
			log.Info("Conflict updating status, will retry")
		} else {
			log.Error(err, "Failed to update status")
		}
	}
}

// ============================================================
// buildDeployment 构建 Deployment 对象
// ============================================================
// 根据 SampleResource Spec 构建 Deployment
//
// 【调用时机】
// reconcileDeployment 中创建 Deployment 时调用

func (r *SampleResourceReconciler) buildDeployment(sr *samplev1alpha1.SampleResource) *appsv1.Deployment {
	replicas := sr.Spec.Replicas
	labels := map[string]string{
		"app":                       sr.Name,
		"app.kubernetes.io/name":   "sample-resource",
		"app.kubernetes.io/instance": sr.Name,
	}
	
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sr.Name,
			Namespace: sr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: sr.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: sr.Spec.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}

// ============================================================
// buildService 构建 Service 对象
// ============================================================
// 根据 SampleResource Spec 构建 Service
//
// 【调用时机】
// reconcileService 中创建 Service 时调用

func (r *SampleResourceReconciler) buildService(sr *samplev1alpha1.SampleResource) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sr.Name,
			Namespace: sr.Namespace,
			Labels: map[string]string{
				"app": sr.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": sr.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       sr.Spec.Port,
					TargetPort: intstr.FromInt(int(sr.Spec.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

// ============================================================
// SetupWithManager 注册控制器到 Manager
// ============================================================
// 配置控制器监听的资源类型和选项
//
// 【调用时机】
// main.go 中初始化 Manager 后调用

func (r *SampleResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// 初始化事件记录器
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor(controllerName)
	}
	
	return ctrl.NewControllerManagedBy(mgr).
		// 监听 SampleResource 资源
		For(&samplev1alpha1.SampleResource{}).
		// 监听 Deployment 资源变化（OwnerReference 关联）
		Owns(&appsv1.Deployment{}).
		// 监听 Service 资源变化（OwnerReference 关联）
		Owns(&corev1.Service{}).
		// 配置选项
		Complete(r)
}
