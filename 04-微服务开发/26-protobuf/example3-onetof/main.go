package main

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	advpb "github.com/go-book/protobuf/proto/advanced"
)

// 演示高级类型：oneof（多选一）、optional（区分未设置/零值）、
// google.protobuf.Timestamp、枚举、嵌套消息、map。
func main() {
	paidAt := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)

	// 用 oneof 表达"支付方式是三选一"
	p := &advpb.Payment{
		OrderId:     "NO-001",
		AmountCents: 19900,
		PaidAt:      timestamppb.New(paidAt),
		Status:      advpb.Payment_STATUS_PAID,
		Method: &advpb.Payment_PayCard{
			PayCard: &advpb.Payment_Card{Last4: "4242", Brand: "visa"},
		},
		Remark:   strPtr("周末促销"),
		Metadata: map[string]string{"device": "ios", "ip": "10.0.0.1"},
	}
	describe(p)

	// 换成另一种支付方式，前一个会被覆盖（oneof 保证同时只有一个）
	p.Method = &advpb.Payment_PayPoints{PayPoints: &advpb.Payment_Points{PointsUsed: 500}}
	describe(p)

	// optional 字段：GetRemark() 返回 *string，能区分"没传"和"传了空串"
	empty := &advpb.Payment{OrderId: "NO-002"}
	fmt.Printf("\nno remark: remark=%v isSet=%v\n",
		empty.GetRemark(), empty.Remark != nil)
	empty.Remark = strPtr("")
	fmt.Printf("empty remark set: remark=%q isSet=%v\n",
		empty.GetRemark(), empty.Remark != nil)

	// Timestamp 转回 time.Time
	fmt.Printf("paid at: %s\n", p.GetPaidAt().AsTime().Format(time.RFC3339))
}

func describe(p *advpb.Payment) {
	switch m := p.Method.(type) {
	case *advpb.Payment_PayCard:
		fmt.Printf("card: %s/%s\n", m.PayCard.Brand, m.PayCard.Last4)
	case *advpb.Payment_PayWechat:
		fmt.Printf("wechat: %s\n", m.PayWechat.TransactionId)
	case *advpb.Payment_PayPoints:
		fmt.Printf("points used: %d\n", m.PayPoints.PointsUsed)
	default:
		fmt.Println("no payment method set")
	}
}

func strPtr(s string) *string { return &s }
