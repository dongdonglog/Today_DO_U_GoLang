package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// 配置回滚演示:利用 etcd 的 MVCC 多版本特性。
//
// 每次修改配置都会产生新的 mod_revision;在 compaction 之前,
// 可以用 WithRev 读到任意历史版本的值 —— 天然的配置版本库。
//
//	go run . -role=init      # 写入 v1 配置
//	go run . -role=break     # 发布一个"坏"的 v2(比如阈值改错导致线上抖动)
//	go run . -role=show      # 列出当前值与全部历史版本
//	go run . -role=rollback  # 从历史 revision 找到上一个版本并恢复
type AppConfig struct {
	RateLimit int    `json:"rate_limit"`
	Note      string `json:"note"`
}

const key = "demo/roll-config"

func main() {
	role := "show"
	for _, a := range os.Args[1:] {
		if len(a) > 6 && a[:6] == "-role=" {
			role = a[6:]
		}
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	ctx := context.Background()

	switch role {
	case "init":
		put(ctx, cli, AppConfig{RateLimit: 100, Note: "v1 稳定版"})
	case "break":
		put(ctx, cli, AppConfig{RateLimit: 99999, Note: "v2 手滑改错了"})
	case "rollback":
		rollback(ctx, cli)
	default:
		show(ctx, cli)
	}
}

func put(ctx context.Context, cli *clientv3.Client, cfg AppConfig) {
	raw, _ := json.Marshal(cfg)
	resp, err := cli.Put(ctx, key, string(raw))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("已写入 %s (mod_revision=%d): %s\n", key, resp.Header.Revision, raw)
}

func show(ctx context.Context, cli *clientv3.Client) {
	cur, err := cli.Get(ctx, key)
	if err != nil || len(cur.Kvs) == 0 {
		log.Fatalf("读取失败或不存在: %v", err)
	}
	fmt.Printf("当前(mod_revision=%d): %s\n", cur.Kvs[0].ModRevision, cur.Kvs[0].Value)

	fmt.Println("--- 历史版本(MVCC) ---")
	for rev := cur.Kvs[0].ModRevision; rev >= cur.Kvs[0].CreateRevision; rev-- {
		resp, err := cli.Get(ctx, key, clientv3.WithRev(rev))
		if err != nil || len(resp.Kvs) == 0 {
			continue // 该 revision 时 key 尚未创建
		}
		fmt.Printf("rev=%d: %s\n", resp.Kvs[0].ModRevision, resp.Kvs[0].Value)
	}
}

// rollback 找到"上一个不同内容"的历史版本并恢复。
func rollback(ctx context.Context, cli *clientv3.Client) {
	cur, err := cli.Get(ctx, key)
	if err != nil || len(cur.Kvs) == 0 {
		log.Fatal("当前配置不存在")
	}
	currentVal := string(cur.Kvs[0].Value)

	var prev []byte
	for rev := cur.Kvs[0].ModRevision - 1; rev >= cur.Kvs[0].CreateRevision; rev-- {
		resp, err := cli.Get(ctx, key, clientv3.WithRev(rev))
		if err != nil || len(resp.Kvs) == 0 {
			continue
		}
		if string(resp.Kvs[0].Value) != currentVal { // 跳过同内容的重复写入
			prev = resp.Kvs[0].Value
			break
		}
	}
	if prev == nil {
		log.Fatal("没有可回滚的历史版本(compaction 可能已清理)")
	}

	fmt.Printf("从历史恢复:\n  当前(坏): %s\n  回滚到:   %s\n", currentVal, prev)
	if _, err := cli.Put(ctx, key, string(prev)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("已恢复。注意:恢复本身也是一次新写入、产生新 revision,watch 它的服务会自动热更新")
}
