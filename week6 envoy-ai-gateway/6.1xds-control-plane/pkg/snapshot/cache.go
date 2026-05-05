// Package snapshot 提供 xDS 配置快照缓存管理
//
// 【核心功能】
// - 存储多个版本的配置快照
// - 支持按 Node ID 隔离配置
// - 版本控制与灰度下发
package snapshot

import (
	"fmt"
	"sync"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// ============================================================
// Cache xDS 配置快照缓存管理器
// ============================================================
// 包装 go-control-plane 的 SnapshotCache
// 增加版本管理与灰度控制逻辑
type Cache struct {
	// inner 内部快照缓存
	inner cache.SnapshotCache

	// mu 保护版本历史
	mu sync.RWMutex

	// versionHistory 版本历史记录
	versionHistory []VersionRecord
}

// ============================================================
// 版本记录：追踪配置变更
// ============================================================
type VersionRecord struct {
	// Version 配置版本号
	Version int64

	// NodeID 目标 Envoy 节点
	NodeID string

	// Status 推送状态 (pending/acked/nacked)
	Status string

	// Timestamp 更新时间
	Timestamp string
}

// ============================================================
// 接口实现：快照缓存管理
// ============================================================

// NewCache 创建配置快照缓存
// 【返回】初始化好的 Cache 实例
func NewCache() *Cache {
	return &Cache{
		inner:          cache.NewSnapshotCache(false, cache.IDHash{}, nil),
		versionHistory: make([]VersionRecord, 0),
	}
}

// UpdateSnapshot 更新快照并记录版本历史
// 【调用时机】配置变化时 (Endpoint 增减、路由规则变更)
// 【参数】nodeID Envoy 节点 ID, snap 新配置快照
func (c *Cache) UpdateSnapshot(nodeID string, snap *cache.Snapshot) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// === 步骤1: 设置快照到内部缓存 ===
	if err := c.inner.SetSnapshot(nodeID, snap); err != nil {
		return fmt.Errorf("failed to set snapshot: %w", err)
	}

	// === 步骤2: 记录版本历史 ===
	record := VersionRecord{
		Version:   snap.Version.(int64),
		NodeID:    nodeID,
		Status:    "pending",
		Timestamp: "2026-05-01T10:00:00Z", // Mock: 实际应使用 time.Now()
	}
	c.versionHistory = append(c.versionHistory, record)

	fmt.Printf("[SnapshotCache] Updated: node=%s, version=%d\n", nodeID, snap.Version)
	return nil
}

// GetSnapshot 获取指定 Node 的当前快照
func (c *Cache) GetSnapshot(nodeID string) (*cache.Snapshot, error) {
	return c.inner.GetSnapshot(nodeID)
}

// GetVersionHistory 获取版本历史记录
func (c *Cache) GetVersionHistory() []VersionRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.versionHistory
}

// Rollback 回滚到指定版本
// 【调用时机】配置下发失败、Envoy NACK 时
// 【参数】nodeID 目标节点, targetVersion 目标版本号
func (c *Cache) Rollback(nodeID string, targetVersion int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// === 步骤1: 查找目标版本快照 ===
	// Mock: 实际应从历史快照中查找
	fmt.Printf("[SnapshotCache] Rolling back node=%s to version=%d\n", nodeID, targetVersion)

	// === 步骤2: 更新版本历史状态 ===
	for i := range c.versionHistory {
		if c.versionHistory[i].NodeID == nodeID &&
			c.versionHistory[i].Version == targetVersion {
			c.versionHistory[i].Status = "rolled_back"
			break
		}
	}

	return nil
}

// ============================================================
// Mock 函数：示意性实现
// ============================================================

// MockCreateAISnapshot 创建 AI 网关场景的配置快照
// 生产环境应从 Kubernetes API 获取真实配置
func MockCreateAISnapshot(nodeID string, version int64) *cache.Snapshot {
	// Mock 实现：返回模拟的配置快照
	// 实际应包含:
	// - LDS: AI 网关监听器 (0.0.0.0:8080)
	// - RDS: 模型路由规则 (/v1/chat -> model-service)
	// - CDS: 模型服务集群定义
	// - EDS: 模型服务 Endpoint 列表

	snapshot := &cache.Snapshot{
		Version: version,
	}

	return snapshot
}
