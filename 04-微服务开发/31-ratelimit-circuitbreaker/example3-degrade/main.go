package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
)

// 降级:非关键依赖失败时,用降级方案保住主流程。
//
//	go run .    # 演示订单详情页:推荐服务故障 → 返回兜底推荐,详情页照常 200
//
// 降级不是"吞掉错误",而是分轻重:
//   - 关键链路(下单/支付):失败就失败,要告警
//   - 非关键链路(推荐/足迹/收藏数):失败降级,给兜底数据,主流程不受影响
//
// 三件套搭配:限流挡住自己人、熔断挡住坏下游、降级兜住非关键功能。

func main() {
	// 推荐服务:用熔断器包裹(短超时+快速失败),失败走 fallback
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "recommend-service",
		MaxRequests: 1,
		Timeout:     2 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 2 // 连续 2 次失败就熔断
		},
	})

	recommend := &recommendSvc{fail: true}
	ctx := context.Background()

	getRecommendation := func(ctx context.Context, userID int64) ([]string, error) {
		v, err := cb.Execute(func() (any, error) {
			return recommend.Fetch(ctx, userID)
		})
		if err != nil {
			return nil, err
		}
		return v.([]string), nil
	}

	// 订单详情页组装:主数据 + 推荐(非关键)
	renderOrderDetail := func(userID int64, orderNo string) {
		fmt.Printf("\n===== 渲染订单 %s =====\n", orderNo)

		// 关键数据:订单本体(假设直接查库成功)
		fmt.Printf("  [主数据] 订单 %s 金额 ¥199.00\n", orderNo)

		// 非关键数据:推荐。失败就降级为默认推荐,不影响主流程
		recs, err := getRecommendation(ctx, userID)
		if err != nil {
			fmt.Printf("  [降级]   推荐服务不可用,返回兜底推荐 (err=%v)\n", err)
			recs = []string{"热销TOP", "新品推荐", "猜你喜欢"}
		}
		fmt.Printf("  [推荐]   %v\n", recs)
	}

	// 第 1~3 次:推荐服务故障 → 每次都降级
	for i := 1; i <= 3; i++ {
		renderOrderDetail(42, fmt.Sprintf("NO-%d", i))
	}

	// 第 4 次:熔断已打开,连推荐服务都不调,直接降级(更快)
	fmt.Println("\n(等待熔断器 Timeout...)")
	time.Sleep(3 * time.Second)
	recommend.fail = false

	// 恢复后:推荐服务正常 → 正常返回
	for i := 4; i <= 5; i++ {
		renderOrderDetail(42, fmt.Sprintf("NO-%d", i))
	}
}

type recommendSvc struct{ fail bool }

func (r *recommendSvc) Fetch(ctx context.Context, userID int64) ([]string, error) {
	if r.fail {
		return nil, errors.New("recommend service timeout")
	}
	return []string{"商品A", "商品B", "商品C"}, nil
}
