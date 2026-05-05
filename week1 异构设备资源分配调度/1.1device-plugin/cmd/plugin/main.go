// Device Plugin 入口程序
// 展示如何启动和注册 Device Plugin 到 kubelet
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/example/device-plugin/pkg/device"
	"github.com/example/device-plugin/pkg/plugin"
	"k8s.io/klog/v2"
)

// ============================================================
// Device Plugin 注册流程
// ============================================================
//
// 1. 启动 gRPC 服务端，监听 Unix Socket
// 2. 向 kubelet 注册插件（通过 kubelet.sock）
// 3. 等待 kubelet 连接并调用 ListAndWatch/Allocate
// 4. 监听终止信号，清理并注销插件

func main() {
	klog.Info("启动 GPU Device Plugin...")

	// === 配置 ===
	resourceName := "example.com/gpu"        // 资源名称，Pod 中使用此名称请求设备
	socketPath := "/var/lib/kubelet/plugins/example.com/gpu.sock"
	kubeletSocket := "/var/lib/kubelet/plugins/kubelet.sock"

	// === 初始化插件 ===
	devices := device.DiscoverDevices()
	deviceMap := make(map[string]device.Device)
	for _, dev := range devices {
		deviceMap[dev.ID] = dev
	}

	gpuPlugin := &plugin.GPUDevicePlugin{
		devices:     deviceMap,
		socketPath:  socketPath,
		resourceName: resourceName,
	}

	// === 启动 gRPC 服务 ===
	// 实际实现需要：
	// 1. 创建 gRPC Server
	// 2. 注册 DevicePlugin 服务
	// 3. 监听 Unix Socket
	//
	// 代码示意：
	//   server := grpc.NewServer()
	//   v1beta1.RegisterDevicePluginServer(server, gpuPlugin)
	//   listener, _ := net.Listen("unix", socketPath)
	//   server.Serve(listener)

	klog.InfoS("启动 gRPC 服务", "socket", socketPath)

	// === 向 kubelet 注册插件 ===
	// 实际实现需要通过 kubelet.sock 发送 RegisterRequest
	//
	// 代码示意：
	//   conn, _ := grpc.Dial(kubeletSocket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	//   client := v1beta1.NewRegistrationClient(conn)
	//   client.Register(context.Background(), &v1beta1.RegisterRequest{
	//       Version:      v1beta1.Version,
	//       Endpoint:     filepath.Base(socketPath),
	//       ResourceName: resourceName,
	//   })

	klog.InfoS("向 kubelet 注册插件", "resource", resourceName)

	// === 监听终止信号 ===
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		klog.Info("收到终止信号，清理并退出...")
		cancel()

		// === 清理：删除 socket 文件 ===
		os.Remove(socketPath)

		// === 清理：向 kubelet 注销插件 ===
		// 实际实现：
		//   client.Register(context.Background(), &v1beta1.RegisterRequest{
		//       ResourceName: resourceName,
		//       Endpoint:     filepath.Base(socketPath),
	//   })

		os.Exit(0)
	}()

	// === 等待运行 ===
	<-ctx.Done()
}

// ============================================================
// 编译与部署说明
// ============================================================
//
// 【编译】
//   GOOS=linux GOARCH=amd64 go build -o gpu-device-plugin ./cmd/plugin
//
// 【构建镜像】
//   FROM ubuntu:22.04
//   COPY gpu-device-plugin /usr/local/bin/gpu-device-plugin
//   ENTRYPOINT ["/usr/local/bin/gpu-device-plugin"]
//
// 【部署】
//   作为 DaemonSet 部署到每个 GPU 节点
//   需要：
//   - hostPath 挂载 /var/lib/kubelet/plugins 目录
//   - hostPath 挂载 /dev 目录（设备文件）
//   - hostPath 挂载驱动库目录
//
// 【注意事项】
//   1. Device Plugin Pod 必须运行在 GPU 节点上
//   2. 需要访问 kubelet 的 plugin 目录（通常是特权容器）
//   3. 需要访问宿主机的设备文件和驱动库
//   4. 插件退出时必须清理 socket 文件并注销