// Package configstore 实现基于 etcd 的类型安全配置中心客户端。
//
// 核心能力:
//   - 启动时从 etcd 加载配置(JSON),加载失败可回退本地默认值
//   - watch 配置 key,变更后自动反序列化、校验、原子替换内存中的配置对象
//   - 新配置校验失败时保留上一份"最后可用配置"(last-good),拒绝坏变更生效
//
// 读侧用 atomic.Pointer 做无锁快照读:业务 goroutine 永远拿到完整的旧配置
// 或完整的新配置,不会读到"改了一半"的中间状态。
package configstore

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Validator 校验新配置是否合法;返回 error 则拒绝这次变更。
type Validator[T any] func(cfg *T) error

type Store[T any] struct {
	key      string
	current  atomic.Pointer[T]
	validate Validator[T]
	onChange []func(old, new *T)
	lastGood *T // 最近一次通过校验的配置,用于坏变更回退
}

// New 创建并初始化配置仓库:
//  1. 从 etcd Get 一次全量;
//  2. key 不存在时使用 defaults(并把 defaults 写回 etcd,保证有据可查);
//  3. 启动 watch,后续变更自动热更新。
func New[T any](ctx context.Context, cli *clientv3.Client, key string,
	defaults *T, validate Validator[T]) (*Store[T], error) {

	s := &Store[T]{key: key, validate: validate}

	// 1. 全量加载
	resp, err := cli.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("读取配置 %s: %w", key, err)
	}

	if len(resp.Kvs) == 0 {
		// key 不存在:写入默认值到 etcd(让配置有唯一事实来源)
		raw, _ := json.MarshalIndent(defaults, "", "  ")
		if _, err := cli.Put(ctx, key, string(raw)); err != nil {
			return nil, fmt.Errorf("写入默认配置: %w", err)
		}
		if err := s.apply(defaults); err != nil {
			return nil, fmt.Errorf("校验默认配置: %w", err)
		}
	} else {
		cfg := new(T)
		if err := json.Unmarshal(resp.Kvs[0].Value, cfg); err != nil {
			return nil, fmt.Errorf("解析配置 %s: %w(key=%s val=%q)",
				key, err, string(resp.Kvs[0].Key), resp.Kvs[0].Value)
		}
		if err := s.apply(cfg); err != nil {
			return nil, fmt.Errorf("校验启动配置: %w", err)
		}
	}

	// 2. watch 增量,实现热更新
	go s.watchLoop(cli)
	return s, nil
}

// Get 返回当前配置快照。返回的是指针共享的对象,
// 业务代码应把它当作只读;需要修改请自行深拷贝。
func (s *Store[T]) Get() *T { return s.current.Load() }

// OnChange 注册配置变更回调(如重建连接池)。
func (s *Store[T]) OnChange(fn func(old, new *T)) {
	s.onChange = append(s.onChange, fn)
}

// apply 校验并原子切换配置;失败则保留 last-good 并报错。
func (s *Store[T]) apply(next *T) error {
	if s.validate != nil {
		if err := s.validate(next); err != nil {
			return fmt.Errorf("配置校验失败: %w", err)
		}
	}
	old := s.current.Load()
	s.current.Store(next)
	s.lastGood = next
	for _, fn := range s.onChange {
		fn(old, next)
	}
	return nil
}

// watchLoop 监听配置 key 的 put/delete:
//   - put:解析+校验+原子切换;坏配置打日志、保留旧配置,服务不中断
//   - delete:视为危险操作,忽略并告警(也可以按团队约定选择回滚到默认值)
func (s *Store[T]) watchLoop(cli *clientv3.Client) {
	for {
		ctx := context.Background()
		resp, err := cli.Get(ctx, s.key)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		wch := cli.Watch(ctx, s.key, clientv3.WithRev(resp.Header.Revision+1))
		for ev := range wch {
			for _, e := range ev.Events {
				switch {
				case e.IsCreate() || e.IsModify():
					next := new(T)
					if err := json.Unmarshal(e.Kv.Value, next); err != nil {
						log.Printf("[config] %s 变更解析失败,保留原配置: %v", s.key, err)
						continue
					}
					if err := s.apply(next); err != nil {
						log.Printf("[config] %s 变更被拒绝,继续使用 last-good 配置: %v", s.key, err)
						continue
					}
					log.Printf("[config] %s 已热更新", s.key)
				case e.Type == clientv3.EventTypeDelete:
					log.Printf("[config] 警告: %s 被删除!拒绝应用空配置,保持 last-good", s.key)
				}
			}
		}
		// watch 断开,重连重建
	}
}
