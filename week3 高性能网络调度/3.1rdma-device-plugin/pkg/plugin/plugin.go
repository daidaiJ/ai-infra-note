// Package plugin RDMA Device Plugin核心实现
package plugin

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"rdma-device-plugin/pkg/device"
)

// ============================================================
// 常量定义
// ============================================================

const (
	// PluginName 插件名称
	PluginName = "rdma"

	// ResourceName 资源类型名称
	ResourceName = "rdma/ib"

	// PluginSocketPath Device Plugin socket目录
	PluginSocketPath = "/var/lib/kubelet/device-plugins"

	// HealthCheckInterval 健康检查间隔
	HealthCheckInterval = 10 * time.Second
)

// ============================================================
// 结构体定义
// ============================================================

// RDADevicePlugin RDMA设备插件
type RDADevicePlugin struct {
	// deviceList 设备列表
	deviceList map[string]*device.DeviceInfo

	// socketPath gRPC socket路径
	socketPath string

	// stopCh 停止信号通道
	stopCh chan struct{}

	// grpcServer gRPC服务器
	grpcServer *v1beta1.DevicePluginServer
}

// ============================================================
// 接口实现
// ============================================================

// NewRDADevicePlugin 创建新的Device Plugin
func NewRDADevicePlugin() *RDADevicePlugin {
	return &RDADevicePlugin{
		deviceList: make(map[string]*device.DeviceInfo),
		socketPath: filepath.Join(PluginSocketPath, PluginName+".sock"),
		stopCh:     make(chan struct{}),
	}
}

// Start 启动Device Plugin
func (p *RDADevicePlugin) Start() error {
	// === 步骤1: 发现设备 ===
	devices, err := device.DiscoverDevices()
	if err != nil {
		return fmt.Errorf("failed to discover devices: %v", err)
	}

	// === 步骤2: 注册设备列表 ===
	for _, dev := range devices {
		p.deviceList[dev.ID] = dev
		glog.Infof("Discovered RDMA device: %s (type: %s, numa: %d)", 
			dev.ID, dev.Type, dev.Topology.NUMANode)
	}

	// === 步骤3: 清理旧socket ===
	if err := p.cleanup(); err != nil {
		return err
	}

	// === 步骤4: 启动gRPC服务 ===
	// Mock实现：示意性启动gRPC服务器
	// 生产环境：使用net.Listen + grpc.NewServer
	glog.Infof("Starting Device Plugin gRPC server at %s", p.socketPath)

	// === 步骤5: 注册到kubelet ===
	// Mock实现：示意性注册
	// 生产环境：调用Register接口
	glog.Infof("Registering Device Plugin with kubelet")

	return nil
}

// Stop 停止Device Plugin
func (p *RDADevicePlugin) Stop() {
	close(p.stopCh)
	p.cleanup()
}

// cleanup 清理socket文件
func (p *RDADevicePlugin) cleanup() error {
	if err := os.Remove(p.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ============================================================
// 实现 Device Plugin gRPC接口
// ============================================================

// GetDevicePluginOptions 返回插件选项
func (p *RDADevicePlugin) GetDevicePluginOptions(context.Context, *v1beta1.Empty) (*v1beta1.DevicePluginOptions, error) {
	return &v1beta1.DevicePluginOptions{
		PreStartRequired: false,
	}, nil
}

// ListAndWatch 上报设备列表和状态变化
// 【调用时机】kubelet启动时建立连接，持续接收设备更新
func (p *RDADevicePlugin) ListAndWatch(req *v1beta1.Empty, stream v1beta1.DevicePlugin_ListAndWatchServer) error {
	// === 步骤1: 发送初始设备列表 ===
	glog.Infof("Sending initial device list")
	devices := p.getGRPCDevices()
	if err := stream.Send(&v1beta1.ListAndWatchResponse{Devices: devices}); err != nil {
		return err
	}

	// === 步骤2: 持续监听设备状态变化 ===
	for {
		select {
		case <-p.stopCh:
			return nil

		case <-time.After(HealthCheckInterval):
			// === 步骤3: 定期检查设备健康 ===
			p.updateDeviceHealth()

			// === 步骤4: 发送更新后的设备列表 ===
			devices := p.getGRPCDevices()
			if err := stream.Send(&v1beta1.ListAndWatchResponse{Devices: devices}); err != nil {
				glog.Errorf("Failed to send device update: %v", err)
				return err
			}
		}
	}
}

// Allocate 分配设备并返回配置注入
// 【调用时机】Pod创建时，kubelet调用Allocate获取设备配置
func (p *RDADevicePlugin) Allocate(ctx context.Context, req *v1beta1.AllocateRequest) (*v1beta1.AllocateResponse, error) {
	glog.Infof("Allocate request: %v", req)

	responses := make([]*v1beta1.ContainerAllocateResponse, 0)

	// === 步骤1: 处理每个Container请求 ===
	for _, containerReq := range req.ContainerRequests {
		// === 步骤2: 获取请求的设备ID列表 ===
		deviceIDs := containerReq.DevicesIDs

		// === 步骤3: 生成设备配置 ===
		response := p.generateContainerResponse(deviceIDs)
		responses = append(responses, response)
	}

	return &v1beta1.AllocateResponse{
		ContainerResponses: responses,
	}, nil
}

// PreStartContainer 容器启动前回调（可选）
func (p *RDADevicePlugin) PreStartContainer(context.Context, *v1beta1.PreStartContainerRequest) (*v1beta1.PreStartContainerResponse, error) {
	return &v1beta1.PreStartContainerResponse{}, nil
}

// ============================================================
// 辅助方法
// ============================================================

// getGRPCDevices 获取gRPC格式的设备列表
func (p *RDADevicePlugin) getGRPCDevices() []*v1beta1.Device {
	devices := make([]*v1beta1.Device, 0, len(p.deviceList))

	for _, dev := range p.deviceList {
		health := v1beta1.Unhealthy
		if dev.Health {
			health = v1beta1.Healthy
		}

		devices = append(devices, &v1beta1.Device{
			ID:      dev.ID,
			Health:  health,
			Topology: &v1beta1.TopologyInfo{
				Node: int64(dev.Topology.NUMANode),
			},
		})
	}

	return devices
}

// updateDeviceHealth 更新设备健康状态
func (p *RDADevicePlugin) updateDeviceHealth() {
	for id, dev := range p.deviceList {
		// Mock实现：假设设备一直健康
		// 生产环境：检查设备状态
		dev.Health = device.CheckDeviceHealth(id)
		p.deviceList[id] = dev
	}
}

// generateContainerResponse 生成容器配置注入
func (p *RDADevicePlugin) generateContainerResponse(deviceIDs []string) *v1beta1.ContainerAllocateResponse {
	// ============================================================
	// 设备文件注入
	// ============================================================
	devices := make([]*v1beta1.DeviceSpec, 0)

	for _, deviceID := range deviceIDs {
		// 主设备文件（verbs接口）
		devices = append(devices, &v1beta1.DeviceSpec{
			HostPath:      fmt.Sprintf("/dev/infiniband/%s", deviceID),
			ContainerPath: fmt.Sprintf("/dev/infiniband/%s", deviceID),
			Permissions:   "mrw", // 读、写、mmap
		})

		// umad设备（用户态管理设备）
		devices = append(devices, &v1beta1.DeviceSpec{
			HostPath:      "/dev/infiniband/umad",
			ContainerPath: "/dev/infiniband/umad",
			Permissions:   "rw",
		})

		// issm设备（IB子网管理）
		devices = append(devices, &v1beta1.DeviceSpec{
			HostPath:      "/dev/infiniband/issm",
			ContainerPath: "/dev/infiniband/issm",
			Permissions:   "rw",
		})
	}

	// ============================================================
	// 环境变量注入
	// ============================================================
	envs := map[string]string{
		"RDMA_DEVICE": strings.Join(deviceIDs, ","),
	}

	return &v1beta1.ContainerAllocateResponse{
		Devices: devices,
		Envs:    envs,
	}
}

// ============================================================
// Mock函数
// ============================================================

// MockStartGRPCServer 启动gRPC服务器（示意性实现）
// 生产环境：使用net.Listen + grpc.NewServer
func MockStartGRPCServer(socketPath string) error {
	// Mock实现：示意性启动
	glog.Infof("Mock: Starting gRPC server at %s", socketPath)
	return nil
}

// MockRegisterToKubelet 注册到kubelet（示意性实现）
// 生产环境：调用Register接口向kubelet注册
func MockRegisterToKubelet(socketPath, resourceName string) error {
	// Mock实现：示意性注册
	glog.Infof("Mock: Registering resource %s via socket %s", resourceName, socketPath)
	return nil
}

// ============================================================
// 主函数
// ============================================================

func Run() {
	glog.Infof("Starting RDMA Device Plugin")

	plugin := NewRDADevicePlugin()
	if err := plugin.Start(); err != nil {
		glog.Fatalf("Failed to start plugin: %v", err)
	}

	// 监听退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	glog.Infof("Received termination signal, shutting down")
	plugin.Stop()
}