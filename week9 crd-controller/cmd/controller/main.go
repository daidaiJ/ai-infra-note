// Package main 提供 CRD 控制器的入口程序
//
// 本程序演示了如何使用 controller-runtime 框架构建一个
// 自定义资源控制器，包括：
// - Manager 初始化
// - 控制器注册
// - 指标和日志配置
// - Leader Election 支持
package main

import (
	"flag"
	"os"

	// 导入所有 Kubernetes API 类型
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	samplev1alpha1 "example.com/week9-crd-controller/pkg/apis/sample/v1alpha1"
	"example.com/week9-crd-controller/pkg/controller"
)

// ============================================================
// 全局变量
// ============================================================

var (
	// scheme 运行时类型注册表
	scheme = runtime.NewScheme()
	
	// setupLog 初始化日志
	setupLog = ctrl.Log.WithName("setup")
)

// ============================================================
// 初始化函数
// ============================================================

func init() {
	// 注册 Kubernetes 内置类型
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	
	// 注册自定义资源类型
	utilruntime.Must(samplev1alpha1.AddToScheme(scheme))
}

// ============================================================
// 主函数
// ============================================================

func main() {
	// === 步骤1: 解析配置参数 ===
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)
	
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.Parse()
	
	// === 步骤2: 配置日志 ===
	// 使用 zap 日志提供者
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
	})))
	
	// === 步骤3: 创建 Manager ===
	// Manager 负责创建和运行控制器
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         metricsAddr,
		Port:                       9443,
		HealthProbeBindAddress:     probeAddr,
		LeaderElection:             enableLeaderElection,
		LeaderElectionID:           "sample-controller-leader",
		LeaderElectionNamespace:    os.Getenv("POD_NAMESPACE"),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	
	// === 步骤4: 注册控制器 ===
	if err = (&controller.SampleResourceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SampleResource")
		os.Exit(1)
	}
	
	// === 步骤5: 注册健康检查 ===
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	
	// === 步骤6: 注册就绪检查 ===
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	
	// === 步骤7: 启动 Manager ===
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
