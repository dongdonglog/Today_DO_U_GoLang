package registry

import (
	"context"
	"encoding/json"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Discover 拉取某服务的全部存活实例(一次性)。
// key 布局与官方 endpoints.Manager 一致:<name>/<addr>。
func Discover(ctx context.Context, client *clientv3.Client, name string) ([]ServiceInfo, error) {
	resp, err := client.Get(ctx, name+"/", clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	out := make([]ServiceInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var ep struct {
			Addr string `json:"addr"`
		}
		if json.Unmarshal(kv.Value, &ep) == nil && ep.Addr != "" {
			out = append(out, ServiceInfo{Name: name, Addr: ep.Addr})
		}
	}
	return out, nil
}

// Watcher 持有某服务实例列表,并通过 watch 自动更新。
type Watcher struct {
	name   string
	client *clientv3.Client

	addrs []string
	ch    chan struct{}
}

// NewWatcher 创建 watcher:立即拉取一次全量列表,之后 watch 前缀增量感知。
func NewWatcher(ctx context.Context, client *clientv3.Client, name string) (*Watcher, error) {
	w := &Watcher{name: name, client: client, ch: make(chan struct{}, 1)}
	if err := w.refresh(ctx); err != nil {
		return nil, err
	}
	go w.loop()
	return w, nil
}

// Addrs 返回当前存活的实例地址。
func (w *Watcher) Addrs() []string { return w.addrs }

// Changed 返回变更通知 channel(多次变化会合并成一次通知)。
func (w *Watcher) Changed() <-chan struct{} { return w.ch }

func (w *Watcher) refresh(ctx context.Context) error {
	resp, err := w.client.Get(ctx, w.name+"/", clientv3.WithPrefix())
	if err != nil {
		return err
	}
	var addrs []string
	for _, kv := range resp.Kvs {
		var info ServiceInfo
		if json.Unmarshal(kv.Value, &info) == nil && info.Addr != "" {
			addrs = append(addrs, info.Addr)
		}
	}
	w.addrs = addrs
	return nil
}

// loop 从当前 revision 起 watch;断连自动重建。
func (w *Watcher) loop() {
	for {
		ctx := context.Background()
		resp, err := w.client.Get(ctx, w.name+"/", clientv3.WithPrefix())
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		_ = w.refresh(ctx)
		w.notify()

		dch := w.client.Watch(ctx, w.name+"/",
			clientv3.WithPrefix(),
			clientv3.WithRev(resp.Header.Revision+1))
		for range dch { // 任何 put/delete 都触发;channel 关闭则重连
			_ = w.refresh(ctx)
			w.notify()
		}
	}
}

func (w *Watcher) notify() {
	select {
	case w.ch <- struct{}{}:
	default: // 已有未处理的通知,合并
	}
}
