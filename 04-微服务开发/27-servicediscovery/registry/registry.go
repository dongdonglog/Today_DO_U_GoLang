// Package registry 实现基于 etcd 的服务注册与发现。
//
// 注册端使用 etcd 官方 endpoints.Manager 的 key 布局(<service>/<addr>),
// 与官方 gRPC resolver(naming/resolver)完全兼容——这样发现端可以直接用
// grpc.NewClient("etcd:///<service>", grpc.WithResolvers(...)) 零胶水接入。
//
// 核心机制:
//   - 注册:写入 key <name>/<addr>,值绑定带 TTL 的租约
//   - 保活:KeepAlive 定时续租;进程崩溃后租约到期,etcd 自动删 key
//   - 下线:Deregister 主动撤销租约(优雅停机)
package registry

import (
	"context"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
)

// ServiceInfo 注册到 etcd 的服务信息
type ServiceInfo struct {
	Name string // 服务名,如 user-service
	Addr string // host:port
}

// Registrar 负责注册与保活。进程退出时调用 Deregister 主动下线。
type Registrar struct {
	client  *clientv3.Client
	em      endpoints.Manager
	key     string
	leaseID clientv3.LeaseID
}

// Register 把服务注册到 etcd 并开始保活。
// ttl 是租约时长(秒):进程崩溃后,最多 ttl 秒 etcd 就会自动摘除该实例。
func Register(ctx context.Context, client *clientv3.Client, info ServiceInfo, ttl int64) (*Registrar, error) {
	// 1. 创建租约(TTL 秒)
	grant, err := client.Grant(ctx, ttl)
	if err != nil {
		return nil, fmt.Errorf("创建租约: %w", err)
	}

	// 2. 用官方 endpoints.Manager 写入:key = <name>/<addr>,与官方 resolver 兼容
	em, err := endpoints.NewManager(client, info.Name)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s/%s", info.Name, info.Addr)
	err = em.AddEndpoint(ctx, key, endpoints.Endpoint{Addr: info.Addr},
		clientv3.WithLease(grant.ID)) // 绑定租约:key 生命周期 = 租约生命周期
	if err != nil {
		return nil, fmt.Errorf("写入注册信息: %w", err)
	}

	// 3. 启动自动续租:每 TTL/3 左右续一次,断连自动重试
	keepCh, err := client.KeepAlive(ctx, grant.ID)
	if err != nil {
		return nil, fmt.Errorf("启动续租: %w", err)
	}
	go func() {
		for range keepCh { // 必须消费 channel,续租才会持续;关闭表示 ctx 取消
		}
	}()

	return &Registrar{client: client, em: em, key: key, leaseID: grant.ID}, nil
}

// Deregister 主动下线:删除 endpoint 并撤销租约。
// 用于优雅停机——不等 TTL 自然过期,避免流量继续打到已停止的实例。
func (r *Registrar) Deregister(ctx context.Context) error {
	if err := r.em.DeleteEndpoint(ctx, r.key); err != nil {
		return err
	}
	_, err := r.client.Revoke(ctx, r.leaseID)
	return err
}
