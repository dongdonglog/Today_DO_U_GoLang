// example3-aggregation 演示聚合管道：按用户分组统计订单金额。
//
// SQL 里的 GROUP BY / SUM / HAVING，在 MongoDB 里用聚合管道 $group/$match/$sort。
// 适合"用户行为日志分析、订单汇总"这类读多、对实时性要求不极端的统计查询。
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

type OrderEvent struct {
	OrderID   string    `bson:"order_id"`
	EventType string    `bson:"event_type"`
	UserID    int64     `bson:"user_id"`
	Amount    int64     `bson:"amount"` // 单位：分
	CreatedAt time.Time `bson:"created_at"`
}

// UserSummary 聚合结果结构
type UserSummary struct {
	UserID     int64 `bson:"_id"`
	TotalSpend int64 `bson:"total_spend"`
	OrderCount int64 `bson:"order_count"`
}

type joinedRow struct {
	UserID     int64       `bson:"_id"`
	TotalSpend int64       `bson:"total_spend"`
	User       []userBrief `bson:"user"`
}

type userBrief struct {
	Name  string `bson:"name"`
	Level string `bson:"level"`
}

type dayRow struct {
	Day   time.Time `bson:"_id"`
	Daily int64     `bson:"daily"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(getenv("MONGO_URI", "mongodb://localhost:27017")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database("shop").Collection("order_events_agg")
	coll.Drop(ctx)

	// 灌数据：3 个用户，每个若干订单事件
	now := time.Now()
	var docs []interface{}
	events := []struct {
		uid    int64
		amount int64
		evt    string
	}{
		{10086, 29900, "paid"}, {10086, 89900, "paid"}, {10086, 5000, "refund"},
		{10087, 199900, "paid"}, {10087, 4900, "paid"},
		{10088, 12900, "paid"},
	}
	for i, e := range events {
		docs = append(docs, OrderEvent{
			OrderID:   fmt.Sprintf("ORD-%d", i),
			EventType: e.evt,
			UserID:    e.uid,
			Amount:    e.amount,
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		})
	}
	coll.InsertMany(ctx, docs)
	fmt.Printf("插入 %d 条事件\n\n", len(docs))

	// ---- 聚合：按 user_id 分组，统计成交订单数和总金额 ----
	// 等价 SQL:
	//   SELECT user_id, COUNT(*) AS order_count, SUM(amount) AS total_spend
	//   FROM order_events
	//   WHERE event_type = 'paid'
	//   GROUP BY user_id
	//   ORDER BY total_spend DESC
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "event_type", Value: "paid"}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$user_id"},
			{Key: "order_count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "total_spend", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "total_spend", Value: -1}}}},
	}
	fmt.Println("=== 按用户聚合成交统计 ===")
	cur, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		log.Fatal(err)
	}
	var results []UserSummary
	cur.All(ctx, &results)
	for _, r := range results {
		fmt.Printf("  user=%d  订单=%d 笔  总消费=%.2f 元\n",
			r.UserID, r.OrderCount, float64(r.TotalSpend)/100)
	}

	// ---- $lookup 关联查询（类似 LEFT JOIN）----
	// 假设另有 users 集合，$lookup 可以把订单关联到用户。
	fmt.Println("\n=== $lookup 关联用户集合 ===")
	users := client.Database("shop").Collection("users")
	users.Drop(ctx)
	users.InsertMany(ctx, []interface{}{
		bson.M{"uid": int64(10086), "name": "张三", "level": "gold"},
		bson.M{"uid": int64(10087), "name": "李四", "level": "silver"},
		bson.M{"uid": int64(10088), "name": "王五", "level": "bronze"},
	})
	joinPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "event_type", Value: "paid"}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$user_id"},
			{Key: "total_spend", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "uid"},
			{Key: "as", Value: "user"},
		}}},
	}
	cur2, _ := coll.Aggregate(ctx, joinPipeline)
	var joined []joinedRow
	cur2.All(ctx, &joined)
	for _, r := range joined {
		if len(r.User) == 0 {
			continue
		}
		fmt.Printf("  %s(%s) 消费 %.2f 元\n",
			r.User[0].Name, r.User[0].Level, float64(r.TotalSpend)/100)
	}

	// ---- 时间维度：按天分组 ----
	fmt.Println("\n=== 按天统计成交金额 ===")
	type DayRow struct {
		Day   time.Time `bson:"_id"`
		Daily int64     `bson:"daily"`
	}
	dayPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "event_type", Value: "paid"}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "$dateTrunc", Value: bson.D{
					{Key: "date", Value: "$created_at"},
					{Key: "unit", Value: "day"},
				}},
			}},
			{Key: "daily", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	cur3, _ := coll.Aggregate(ctx, dayPipeline)
	var days []dayRow
	cur3.All(ctx, &days)
	for _, r := range days {
		fmt.Printf("  %s  成交 %.2f 元\n", r.Day.Format("2006-01-02"), float64(r.Daily)/100)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
