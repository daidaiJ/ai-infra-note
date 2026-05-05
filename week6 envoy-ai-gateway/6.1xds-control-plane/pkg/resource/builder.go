// Package resource 提供 xDS 资源构建器
//
// 【核心功能】
// - 构建 LDS (Listener) 监听器资源
// - 构建 RDS (Route) 路由资源
// - 构建 CDS (Cluster) 集群资源
// - 构建 EDS (Endpoint) 端点资源
package resource

import (
	"fmt"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

// ============================================================
// 常量定义：AI 网关资源名称
// ============================================================
const (
	// ListenerName AI 网关监听器名称
	ListenerName = "ai-gateway-listener"

	// RouteConfigName 路由配置名称
	RouteConfigName = "ai-gateway-routes"

	// ClusterModelV1 模型服务 v1 集群名称
	ClusterModelV1 = "model-service-v1"

	// ClusterModelV2 模型服务 v2 集群名称 (金丝雀)
	ClusterModelV2 = "model-service-v2"
)

// ============================================================
// Builder 资源构建器
// ============================================================
// 封装 LDS/RDS/CDS/EDS 资源的创建逻辑
// 用于快速生成 AI 网关场景的配置
type Builder struct{}

// NewBuilder 创建资源构建器
func NewBuilder() *Builder {
	return &Builder{}
}

// ============================================================
// LDS 构建：创建监听器
// ============================================================

// BuildListener 创建 AI 网关监听器
// 【核心字段】
// - Address: 0.0.0.0 (监听所有网卡)
// - Port: 8080 (HTTP 端口)
// - FilterChain: HTTP Connection Manager
func (b *Builder) BuildListener() *listener.Listener {
	// === 步骤1: 配置 HTTP Connection Manager ===
	hcmFilter := &listener.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listener.Filter_TypedConfig{
			TypedConfig: marshalMessage(&hcm.HttpConnectionManager{
				StatPrefix: "ai_gateway",
				CodecType:  hcm.HttpConnectionManager_AUTO,
				RouteSpecifier: &hcm.HttpConnectionManager_Rds{
					Rds: &hcm.Rds{
						ConfigSource: &core.ConfigSource{
							ResourceApiVersion: resource.DefaultAPIVersion,
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
						},
						RouteConfigName: RouteConfigName,
					},
				},
			}),
		},
	}

	// === 步骤2: 创建监听器 ===
	return &listener.Listener{
		Name: ListenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 8080,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{hcmFilter},
			},
		},
	}
}

// ============================================================
// RDS 构建：创建路由规则
// ============================================================

// BuildRouteConfig 创建 AI 模型路由配置
// 【核心路由规则】
// - /v1/chat/completions -> model-service (v1/v2 权重分流)
// - /v1/completions -> model-service-v1
func (b *Builder) BuildRouteConfig() *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: RouteConfigName,
		VirtualHosts: []*route.VirtualHost{
			{
				Name:    "ai-gateway-vhost",
				Domains: []string{"*"},
				Routes: []*route.Route{
					// === 路由1: Chat Completions (支持金丝雀) ===
					{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/v1/chat/completions",
							},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_WeightedClusters{
									WeightedClusters: &route.WeightedCluster{
										Clusters: []*route.WeightedCluster_ClusterWeight{
											{
												Name:   ClusterModelV1,
												Weight: 90, // v1 占 90% 流量
											},
											{
												Name:   ClusterModelV2,
												Weight: 10, // v2 占 10% 流量 (金丝雀)
											},
										},
									},
								},
							},
						},
					},
					// === 路由2: Completions (仅 v1) ===
					{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{
								Prefix: "/v1/completions",
							},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_Cluster{
									Cluster: ClusterModelV1,
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
// CDS 构建：创建集群定义
// ============================================================

// BuildCluster 创建模型服务集群
// 【核心字段】
// - Name: 集群名称
// - ConnectTimeout: 连接超时
// - LbPolicy: 负载均衡策略 (ROUND_ROBIN)
func (b *Builder) BuildCluster(name string) *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 name,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ResourceApiVersion: resource.DefaultAPIVersion,
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
		LbPolicy: cluster.Cluster_ROUND_ROBIN,
	}
}

// ============================================================
// EDS 构建：创建端点列表
// ============================================================

// BuildEndpoint 创建模型服务端点列表
// 【参数】clusterName 集群名称, endpoints 端点地址列表
// 【返回】Endpoint 资源
func (b *Builder) BuildEndpoint(clusterName string, endpoints []string) *endpoint.ClusterLoadAssignment {
	var lbs []*endpoint.LbEndpoint

	for _, addr := range endpoints {
		lbs = append(lbs, &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  addr,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: 8000, // 模型服务端口
								},
							},
						},
					},
				},
			},
		})
	}

	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: lbs,
			},
		},
	}
}

// ============================================================
// Mock 函数：示意性实现
// ============================================================

// MockBuildAIResources 构建 AI 网关场景的完整资源配置
// 生产环境应从 Kubernetes API 获取真实配置
func MockBuildAIResources() map[string]proto.Message {
	// Mock 实现：返回模拟的资源集合
	// 实际应包含完整的 LDS/RDS/CDS/EDS 资源

	resources := map[string]proto.Resource{}

	return resources
}
