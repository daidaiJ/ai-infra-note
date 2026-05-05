// 自定义 GPU 感知调度器入口
// 展示如何注册自定义插件并启动调度器
package main

import (
	"os"

	"github.com/example/gpu-scheduler/pkg/plugins/deviceuuid"
	"github.com/example/gpu-scheduler/pkg/plugins/gpuutilization"
	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	// 内置插件必须导入以触发 init() 注册
	_ "k8s.io/kubernetes/pkg/scheduler/framework/plugins" // 所有内置插件
)

func main() {
	// ============================================================
	// 创建调度器命令
	// ============================================================
	//
	// app.NewSchedulerCommand 是 K8s 官方提供的调度器入口
	// 通过 app.WithPlugin 选项注册自定义插件
	//
	// 插件注册后，可在配置文件中通过名称启用和配置
	//
	command := app.NewSchedulerCommand(
		// 注册 GPU 利用率均衡插件
		app.WithPlugin(gpuutilization.Name, gpuutilization.New),

		// 注册设备 UUID 匹配插件
		app.WithPlugin(deviceuuid.Name, deviceuuid.New),
	)

	// 初始化日志
	logs.InitLogs()
	defer logs.FlushLogs()

	// 执行调度器
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}

// ============================================================
// 编译与部署说明
// ============================================================
//
// 【编译自定义调度器】
//   GOOS=linux GOARCH=amd64 go build -o gpu-scheduler ./cmd/scheduler
//
// 【构建容器镜像】
//   FROM gcr.io/distroless/static:latest
//   COPY gpu-scheduler /usr/local/bin/kube-scheduler
//   ENTRYPOINT ["/usr/local/bin/kube-scheduler"]
//
// 【运行方式】
//   1. 替换默认调度器: 使用自定义镜像替换 kube-scheduler 镜像
//   2. 并行运行: 作为独立调度器，通过 schedulerName 区分
//
// 【配置文件】
//   通过 --config 参数指定调度器配置文件
//   见 config/scheduler-config.yaml