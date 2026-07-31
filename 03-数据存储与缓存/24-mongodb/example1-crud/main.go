// example1-crud 演示 MongoDB 基本 CRUD：连接、插入、查询、更新、删除。
//
// 场景：订单系统里的"订单变更历史"——每条记录结构不完全一样（不同事件有不同字段），
// 用文档模型比关系表更自然。
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

// OrderEvent 订单事件。字段通过 bson tag 映射到文档字段。
type OrderEvent struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	OrderID   string        `bson:"order_id"`
	EventType string        `bson:"event_type"` // created / paid / shipped / cancelled
	UserID    int64         `bson:"user_id"`
	Amount    int64         `bson:"amount"`
	CreatedAt time.Time     `bson:"created_at"`
	// 不同事件的扩展字段用 map 存，正好体现文档模型的灵活性
	Extra map[string]interface{} `bson:"extra,omitempty"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := getenv("MONGO_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("连接失败: %v\n启动前请执行:\n  docker run --name go-book-mongo -p 27017:27017 -d mongo:7", err)
	}
	defer client.Disconnect(ctx)
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Ping 失败: %v", err)
	}
	fmt.Println("MongoDB 连接成功")

	coll := client.Database("shop").Collection("order_events")
	coll.Drop(ctx) // 示例每次重置

	// ---- 1. 插入 ----
	fmt.Println("\n=== 插入单条 ===")
	now := time.Now()
	evt := OrderEvent{
		OrderID:   "ORD-1001",
		EventType: "paid",
		UserID:    10086,
		Amount:    29900,
		CreatedAt: now,
		Extra:     map[string]interface{}{"pay_method": "alipay", "txn_id": "20241001xxx"},
	}
	insertRes, err := coll.InsertOne(ctx, evt)
	if err != nil {
		log.Fatal(err)
	}
	objID := insertRes.InsertedID.(bson.ObjectID)
	fmt.Printf("插入成功 _id=%s\n", objID.Hex())

	// ---- 2. 查询 ----
	fmt.Println("\n=== 查询 ===")
	var got OrderEvent
	if err := coll.FindOne(ctx, bson.M{"order_id": "ORD-1001"}).Decode(&got); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("查到: %s 事件=%s 金额=%d extra.pay_method=%v\n",
		got.OrderID, got.EventType, got.Amount, got.Extra["pay_method"])

	// ---- 3. 更新 ----
	fmt.Println("\n=== 更新 ===")
	// $set 更新部分字段；mongo 不会像 MySQL 行锁那样全记录覆盖
	coll.UpdateOne(ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"amount": 25900, "extra.coupon": "NEW10"}},
	)
	var after OrderEvent
	coll.FindOne(ctx, bson.M{"_id": objID}).Decode(&after)
	fmt.Printf("更新后: 金额=%d extra.coupon=%v\n", after.Amount, after.Extra["coupon"])

	// ---- 4. 批量插入 ----
	fmt.Println("\n=== 批量插入 3 条 ===")
	docs := []interface{}{
		OrderEvent{OrderID: "ORD-1002", EventType: "created", UserID: 10086, Amount: 89900, CreatedAt: now.Add(-time.Hour)},
		OrderEvent{OrderID: "ORD-1003", EventType: "shipped", UserID: 10087, Amount: 199900, CreatedAt: now.Add(-2 * time.Hour), Extra: map[string]interface{}{"tracking": "SF123"}},
		OrderEvent{OrderID: "ORD-1004", EventType: "cancelled", UserID: 10086, Amount: 5000, CreatedAt: now.Add(-3 * time.Hour)},
	}
	res, _ := coll.InsertMany(ctx, docs)
	fmt.Printf("插入 %d 条\n", len(res.InsertedIDs))

	// ---- 5. 条件查询 + 排序 + 分页 ----
	fmt.Println("\n=== 查询 user_id=10086 的事件，按时间倒序 ===")
	cur, err := coll.Find(ctx,
		bson.M{"user_id": 10086},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(10),
	)
	if err != nil {
		log.Fatal(err)
	}
	var list []OrderEvent
	cur.All(ctx, &list)
	for _, e := range list {
		fmt.Printf("  %s | %s | %d 分\n", e.CreatedAt.Format("15:04:05"), e.EventType, e.Amount)
	}

	// ---- 6. 删除 ----
	fmt.Println("\n=== 删除 ===")
	delRes, _ := coll.DeleteOne(ctx, bson.M{"_id": objID})
	fmt.Printf("删除 %d 条\n", delRes.DeletedCount)
	count, _ := coll.CountDocuments(ctx, bson.D{})
	fmt.Printf("剩余文档数: %d\n", count)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
