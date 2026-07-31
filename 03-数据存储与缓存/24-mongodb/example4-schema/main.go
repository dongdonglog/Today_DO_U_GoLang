// example4-schema 演示文档模型的灵活性：不同事件类型有不同字段，
// 以及嵌入式文档避免多表 JOIN。
//
// 这正是 MongoDB 相对关系型数据库的核心优势：schema-on-read，
// 结构不确定的数据（订单日志、用户行为、审计事件）无需提前设计表结构。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AuditEvent 审计事件。不同事件类型 detail 字段结构完全不同，
// 用 interface{} 承载，这是关系型里要设计多表/JSON 列才能做到的事。
type AuditEvent struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    int64         `bson:"user_id"`
	Action    string        `bson:"action"` // login / order_create / refund / profile_update
	Timestamp time.Time     `bson:"ts"`
	Detail    interface{}   `bson:"detail"` // 不同 action 对应不同结构
}

type LoginDetail struct {
	IP      string `bson:"ip"`
	Device  string `bson:"device"`
	Success bool   `bson:"success"`
}

type OrderDetail struct {
	OrderID string `bson:"order_id"`
	Amount  int64  `bson:"amount"`
	Items   int    `bson:"items"`
}

type RefundDetail struct {
	OrderID string `bson:"order_id"`
	Reason  string `bson:"reason"`
}

type ProfileDetail struct {
	ChangedFields []string `bson:"changed_fields"`
}

// Order 订单用嵌入式文档存"订单头+明细"，关系型里需要 orders + order_items 两张表。
type Order struct {
	ID      bson.ObjectID `bson:"_id,omitempty"`
	OrderNo string        `bson:"order_no"`
	UserID  int64         `bson:"user_id"`
	Amount  int64         `bson:"amount"`
	Status  string        `bson:"status"`
	// 明细直接嵌入数组，一次查询把订单和所有明细都拿出来，无需 JOIN
	Items []OrderItem `bson:"items"`
}

type OrderItem struct {
	SKU   string `bson:"sku"`
	Name  string `bson:"name"`
	Qty   int    `bson:"qty"`
	Price int64  `bson:"price"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(getenv("MONGO_URI", "mongodb://localhost:27017")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("shop")

	// ---- 1. 异构事件：不同 action 的 detail 字段不同 ----
	fmt.Println("=== 不同类型的审计事件，字段各不相同 ===")
	events := db.Collection("audit_events")
	events.Drop(ctx)

	docs := []interface{}{
		AuditEvent{UserID: 10086, Action: "login", Timestamp: time.Now(),
			Detail: LoginDetail{IP: "1.2.3.4", Device: "iOS", Success: true}},
		AuditEvent{UserID: 10086, Action: "order_create", Timestamp: time.Now(),
			Detail: OrderDetail{OrderID: "ORD-1", Amount: 29900, Items: 2}},
		AuditEvent{UserID: 10087, Action: "refund", Timestamp: time.Now(),
			Detail: RefundDetail{OrderID: "ORD-2", Reason: "尺寸不符"}},
		AuditEvent{UserID: 10088, Action: "profile_update", Timestamp: time.Now(),
			Detail: ProfileDetail{ChangedFields: []string{"phone", "address"}}},
	}
	events.InsertMany(ctx, docs)

	// 按 action 过滤查询，不需要提前建表
	type loginRow struct {
		UserID int64       `bson:"user_id"`
		Detail LoginDetail `bson:"detail"`
	}
	cur, _ := events.Find(ctx, bson.M{"action": "login"})
	var logins []loginRow
	cur.All(ctx, &logins)
	for _, l := range logins {
		fmt.Printf("  login: user=%d ip=%s\n", l.UserID, l.Detail.IP)
	}

	// ---- 2. 嵌入式文档：订单 + 明细 ----
	fmt.Println("\n=== 嵌入式文档：订单+明细，一次查询全部返回 ===")
	orders := db.Collection("orders")
	orders.Drop(ctx)

	order := Order{
		OrderNo: "ORD-20241001-001",
		UserID:  10086,
		Amount:  32800,
		Status:  "paid",
		Items: []OrderItem{
			{SKU: "KB-001", Name: "机械键盘", Qty: 1, Price: 25900},
			{SKU: "MP-009", Name: "鼠标垫", Qty: 1, Price: 6900},
		},
	}
	orders.InsertOne(ctx, order)

	var got Order
	orders.FindOne(ctx, bson.M{"order_no": "ORD-20241001-001"}).Decode(&got)
	fmt.Printf("  订单: %s  总金额: %d 分\n", got.OrderNo, got.Amount)
	for _, item := range got.Items {
		fmt.Printf("    - %s x %d = %d 分\n", item.Name, item.Qty, item.Price*int64(item.Qty))
	}

	// 可以直接对嵌入式数组字段做查询和索引（点表示法）
	fmt.Println("\n=== 查询包含 SKU=KB-001 的所有订单 ===")
	cur2, _ := orders.Find(ctx, bson.M{"items.sku": "KB-001"})
	var matches []Order
	cur2.All(ctx, &matches)
	for _, m := range matches {
		fmt.Printf("  命中: %s\n", m.OrderNo)
	}

	// ---- 3. 字段稀疏：不是每个文档都有全部字段 ----
	// 关系型里要么 NULL 要么要额外关联表，Mongo 里不存在的字段就不占空间
	fmt.Println("\n=== 稀疏字段：只有 paid 事件有 pay_method ===")
	fmt.Println("（无需 ALTER TABLE，直接加字段即可）")
	events.InsertOne(ctx, bson.M{
		"user_id":    int64(10086),
		"action":     "pay_success",
		"ts":         time.Now(),
		"pay_method": "alipay", // 这个字段只在 pay_success 里有
		"amount":     29900,
	})
	count, _ := events.CountDocuments(ctx, bson.M{"pay_method": "alipay"})
	fmt.Printf("  pay_method=alipay 的事件数: %d\n", count)
	count2, _ := events.CountDocuments(ctx, bson.M{"pay_method": bson.M{"$exists": true}})
	fmt.Printf("  有 pay_method 字段的文档: %d / 总 %d\n", count2, len(docs)+1)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
