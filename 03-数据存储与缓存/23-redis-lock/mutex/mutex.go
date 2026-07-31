// Package mutex 提供基于 Redis 的分布式互斥锁实现。
//
// 设计要点（对应正文章节）：
//   - 原子加锁：SET key value NX PX ttl，一次网络往返完成"不存在才加"，
//     避免 SETNX+EXPIRE 两步之间进程崩溃导致的死锁（见 23.2）。
//   - 唯一持有者标识：value 使用 UUID，释放时只允许锁的持有者释放，
//     防止"过期后被别人拿到、你再 DEL 把别人的锁删了"的误释放（见 23.3）。
//   - Lua 原子释放：GET 判断 + DEL 必须在 Redis 服务端原子执行，
//     避免判断与删除之间被其他命令插入（见 23.3）。
//   - 看门狗续期：持有锁期间自动续约，避免业务执行时间超过 TTL 导致锁自动过期（见 23.4）。
package mutex

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrLockNotAcquired 表示加锁失败（已被别人持有），区别于网络错误。
var ErrLockNotAcquired = errors.New("lock not acquired")

// ErrLockNotHeld 表示释放/续约时发现锁已不属于自己（可能已过期或被误释放）。
var ErrLockNotHeld = errors.New("lock not held by current owner")

// 默认值，调用方通过 Option 覆盖。
const (
	defaultTTL      = 10 * time.Second
	defaultRenew    = 3 * time.Second // 续约间隔，取 TTL 的 1/3 左右
	defaultMaxRetry = 0               // 默认不重试，立即返回；业务层决定重试策略
)

// Mutex 分布式锁，一个实例对应一个 key，不直接跨 key 复用。
type Mutex struct {
	rdb   *redis.Client
	key   string
	ttl   time.Duration
	renew time.Duration
	owner string // 持有者唯一标识，释放时校验
}

// Option 配置 Mutex。
type Option func(*Mutex)

// WithTTL 设置锁的过期时间（业务最长执行时间的 2 倍左右，留余量）。
func WithTTL(d time.Duration) Option {
	return func(m *Mutex) { m.ttl = d }
}

// WithRenewInterval 设置看门狗续约间隔，默认 TTL/3。
func WithRenewInterval(d time.Duration) Option {
	return func(m *Mutex) { m.renew = d }
}

// New 构造一把 Redis 分布式锁。
func New(rdb *redis.Client, key string, opts ...Option) *Mutex {
	m := &Mutex{
		rdb:   rdb,
		key:   key,
		ttl:   defaultTTL,
		renew: 0, // 0 表示在 New 结束后按 ttl/3 计算
		owner: randomToken(),
	}
	for _, o := range opts {
		o(m)
	}
	if m.renew == 0 {
		m.renew = m.ttl / 3
		if m.renew < 100*time.Millisecond {
			m.renew = 100 * time.Millisecond
		}
	}
	return m
}

// randomToken 生成 16 字节随机 hex 作为持有者标识，碰撞概率可忽略。
func randomToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Lock 尝试加锁；失败立即返回 ErrLockNotAcquired（不阻塞）。
// 要阻塞/重试请在业务层循环调用（配合 sleep/退避），或使用 LockContext。
func (m *Mutex) Lock(ctx context.Context) error {
	ok, err := m.rdb.SetNX(ctx, m.key, m.owner, m.ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockNotAcquired
	}
	return nil
}

// unlockScript 原子释放：先 GET 判断持有者一致再 DEL。
// KEYS[1]=锁key, ARGV[1]=owner token
// 返回 1 表示释放成功，0 表示锁已不属于自己。
const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end
`

// Unlock 释放锁，只有持有者才能释放。
func (m *Mutex) Unlock(ctx context.Context) error {
	res, err := m.rdb.Eval(ctx, unlockScript, []string{m.key}, m.owner).Result()
	if err != nil {
		return err
	}
	if n, _ := res.(int64); n != 1 {
		return ErrLockNotHeld
	}
	return nil
}

// renewScript 原子续约：持有者一致才 PEXPIRE 延长 TTL。
// KEYS[1]=锁key, ARGV[1]=owner, ARGV[2]=新 TTL(毫秒)
const renewScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("pexpire", KEYS[1], ARGV[2])
else
	return 0
end
`

// Renew 手动续约。通常不需要直接调用，用 RunWatchdog 自动续。
func (m *Mutex) Renew(ctx context.Context) error {
	ms := int64(m.ttl / time.Millisecond)
	res, err := m.rdb.Eval(ctx, renewScript, []string{m.key}, m.owner, ms).Result()
	if err != nil {
		return err
	}
	if n, _ := res.(int64); n != 1 {
		return ErrLockNotHeld
	}
	return nil
}

// RunWatchdog 在后台定期续约，返回一个 stop 函数。
// 调用方在业务结束时（或 defer 里）调用 stop() 停止续约；
// 锁被意外释放（过期/被抢占）时续约失败，自动停止并通知 errCh。
func (m *Mutex) RunWatchdog(ctx context.Context) (stop func(), errCh <-chan error) {
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan error, 1)

	go func() {
		ticker := time.NewTicker(m.renew)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.Renew(ctx); err != nil {
					ch <- err
					return
				}
			}
		}
	}()

	return cancel, ch
}
