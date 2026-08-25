package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/sony/gobreaker"
)

// 熔断:保护自己不反复调用已故障的下游。
//
//	go run .    # 内置场景:下游先故障→熔断打开→快速失败→恢复后半开试探
//
// 状态机:
//   Closed(关闭)    :正常调用,统计失败率;失败率超阈值 → Open
//   Open(打开)      :直接快速失败,不调下游;等 timeout 冷却 → Half-Open
//   Half-Open(半开) :放一个试探请求;成功→Closed,失败→Open
//
// gobreaker 的失败只认它看得到的 error:业务逻辑错误(如"库存不足")
// 不应计入熔断——那些不是"下游故障"。用 gobreaker.IsSuccessful 显式声明。

type downstream struct{ healthy bool }

// Call 模拟调用下游:故障时返回错误,恢复正常时成功
func (d *downstream) Call(ctx context.Context, req string) (string, error) {
	if !d.healthy {
		return "", errors.New("下游超时/5xx")
	}
	// 恢复初期有小概率仍失败,模拟半开试探期的抖动
	if rand.Intn(100) < 20 {
		return "", errors.New("瞬时抖动")
	}
	return "resp[" + req + "]", nil
}

func main() {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "payment-downstream",
		MaxRequests: 1,                // 半开时放 1 个试探请求
		Interval:    10 * time.Second, // 统计窗口
		Timeout:     3 * time.Second,  // Open 多久后进 Half-Open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.6 // 5 个请求且 60% 失败 → Open
		},
		// 除业务错误外的失败才计入熔断统计:业务错误(如"库存不足")不是下游故障
		IsSuccessful: func(err error) bool {
			return err == nil || errors.Is(err, errBusiness)
		},
	})

	down := &downstream{healthy: true}

	call := func(i int) {
		state := cb.State()
		result, err := cb.Execute(func() (any, error) {
			return down.Call(context.Background(), fmt.Sprintf("req-%d", i))
		})
		if err != nil {
			fmt.Printf("[%02d] %-11s 失败: %v\n", i, state.String(), err)
			return
		}
		fmt.Printf("[%02d] %-11s 成功: %v\n", i, state.String(), result)
	}

	fmt.Println("=== 阶段1:下游正常(第1~5个请求) ===")
	for i := 1; i <= 5; i++ {
		call(i)
	}

	fmt.Println("\n=== 阶段2:下游故障 → 熔断打开 → 快速失败(第6~20个请求) ===")
	down.healthy = false
	for i := 6; i <= 20; i++ {
		call(i)
		if i%5 == 0 {
			fmt.Printf("      当前状态=%s\n", cb.State().String())
		}
	}

	fmt.Println("\n=== 阶段3:下游恢复,等待 Timeout 后进入半开试探(第21~30个请求) ===")
	down.healthy = true
	time.Sleep(4 * time.Second) // 等超过 Timeout=3s,Open → Half-Open
	for i := 21; i <= 30; i++ {
		call(i)
		if i%5 == 0 {
			fmt.Printf("      当前状态=%s\n", cb.State().String())
		}
	}
}

// errBusiness 表示业务层面的失败(不计入熔断统计)
var errBusiness = errors.New("业务错误:库存不足")

var _ = log.Println // 保留占位避免 lint 告警
