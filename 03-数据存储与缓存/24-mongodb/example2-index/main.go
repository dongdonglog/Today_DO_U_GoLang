// example2-index 演示 MongoDB 索引：创建、查看、执行计划。
//
// 索引是 MongoDB 性能的第一抓手——没有合适的索引，集合一大就是全表扫描(COLLSCAN)，
// 和 MySQL 没建索引是同一个道理。
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
	ID        bson.ObjectID          `bson:"_id,omitempty"`
	OrderID   string                 `bson:"order_id"`
	EventType string                 `bson:"event_type"`
	UserID    int64                  `bson:"user_id"`
	Amount    int64                  `bson:"amount"`
	CreatedAt time.Time              `bson:"created_at"`
	Extra     map[string]interface{} `bson:"extra,omitempty"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	uri := getenv("MONGO_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database("shop").Collection("order_events_idx")
	coll.Drop(ctx)

	// ---- 1. 先灌入 5000 条测试数据 ----
	fmt.Println("=== 灌入 5000 条测试数据 ===")
	now := time.Now()
	var docs []interface{}
	for i := 0; i < 5000; i++ {
		docs = append(docs, OrderEvent{
			OrderID:   fmt.Sprintf("ORD-%05d", i),
			EventType: []string{"created", "paid", "shipped", "cancelled"}[i%4],
			UserID:    int64(10000 + i%100),
			Amount:    int64(1000 + i*10),
			CreatedAt: now.Add(-time.Duration(i) * time.Minute),
		})
	}
	res, _ := coll.InsertMany(ctx, docs)
	fmt.Printf("插入 %d 条\n", len(res.InsertedIDs))

	// ---- 2. 没索引时的执行计划 ----
	fmt.Println("\n=== 没索引：全表扫描 ===")
	explainNoIdx := explain(ctx, client, coll, bson.D{{Key: "user_id", Value: int64(10050)}})
	fmt.Printf("stage=%s（无索引时遍历整个集合）\n", explainNoIdx)

	// ---- 3. 创建单字段索引 ----
	fmt.Println("\n=== 创建单字段索引：user_id ===")
	_, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}}, // 1=升序，-1=降序，单字段方向不影响
	})
	if err != nil {
		log.Fatal(err)
	}
	explainUserIdx := explain(ctx, client, coll, bson.D{{Key: "user_id", Value: int64(10050)}})
	fmt.Printf("stage=%s（命中 user_id 索引）\n", explainUserIdx)

	// ---- 4. 复合索引：user_id + created_at ----
	// 查 "某用户最近的事件" 这种高频查询，单字段索引不够用。
	// 复合索引遵循 ESR 规则：等值(Equality)→排序(Sort)→范围(Range)
	fmt.Println("\n=== 创建复合索引：user_id + created_at 降序 ===")
	_, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	cursor, _ := coll.Find(ctx,
		bson.D{{Key: "user_id", Value: int64(10050)}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(3),
	)
	var results []OrderEvent
	cursor.All(ctx, &results)
	for _, r := range results {
		fmt.Printf("  %s | %s | %d\n", r.CreatedAt.Format("15:04"), r.EventType, r.Amount)
	}
	// 不带 sort 也能命中复合索引；这里仅验证存在性
	explainCompound := explain(ctx, client, coll, bson.D{{Key: "user_id", Value: int64(10050)}})
	fmt.Printf("复合索引存在后 stage=%s\n", explainCompound)

	// ---- 5. 唯一索引：防重复 ----
	fmt.Println("\n=== 唯一索引：order_id 不重复 ===")
	_, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "order_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	fmt.Printf("唯一索引创建: err=%v\n", err)
	_, err = coll.InsertOne(ctx, OrderEvent{OrderID: "ORD-00001", EventType: "paid"})
	if mongo.IsDuplicateKeyError(err) {
		fmt.Println("重复插入正确报错：duplicate key")
	}

	// ---- 6. TTL 索引：自动过期（和 Redis TTL 类似）----
	fmt.Println("\n=== TTL 索引：created_at 1 小时自动过期（这里仅创建，演示用）===")
	// expireAfterSeconds=0 表示字段本身就是过期时间点；设 >0 表示"字段时间 + N 秒"过期
	_, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "created_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(3600),
	})
	fmt.Printf("TTL 索引创建: err=%v\n", err)

	// ---- 7. 列出所有索引 ----
	fmt.Println("\n=== 集合所有索引 ===")
	cur, _ := coll.Indexes().List(ctx)
	for cur.Next(ctx) {
		var idx bson.M
		cur.Decode(&idx)
		fmt.Printf("  name=%v key=%v unique=%v\n", idx["name"], idx["key"], idx["unique"])
	}
}

// explain 通过 explain 命令返回查询计划的叶子 stage 名称（COLLSCAN / IXSCAN）。
func explain(ctx context.Context, client *mongo.Client, coll *mongo.Collection, filter bson.D) string {
	cmd := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: coll.Name()},
			{Key: "filter", Value: filter},
		}},
	}
	var result bson.M
	if err := client.Database(coll.Database().Name()).RunCommand(ctx, cmd).Decode(&result); err != nil {
		return err.Error()
	}
	// bson.D → bson.M 转换，保证后续类型断言统一
	es, _ := toM(result["executionStats"])
	if es != nil {
		stages, _ := toM(es["executionStages"])
		if stages != nil {
			return findLeafStage(stages)
		}
	}
	qp, _ := toM(result["queryPlanner"])
	if qp != nil {
		wp, _ := toM(qp["winningPlan"])
		if wp != nil {
			return findLeafStage(wp)
		}
	}
	return "unknown"
}

// toM 把 bson.D / bson.M / map[string]interface{} 统一成 bson.M。
func toM(v interface{}) (bson.M, bool) {
	switch x := v.(type) {
	case bson.M:
		return x, true
	case map[string]interface{}:
		return bson.M(x), true
	case bson.D:
		m := bson.M{}
		for _, e := range x {
			m[e.Key] = e.Value
		}
		return m, true
	default:
		return nil, false
	}
}

func findLeafStage(plan bson.M) string {
	if in, ok := plan["inputStage"]; ok {
		if m, ok2 := toM(in); ok2 {
			if res := findLeafStage(m); res != "" {
				return res
			}
		}
	}
	if arr, ok := plan["inputStages"].(bson.A); ok && len(arr) > 0 {
		if m, ok2 := toM(arr[0]); ok2 {
			if res := findLeafStage(m); res != "" {
				return res
			}
		}
	}
	if s, ok := plan["stage"].(string); ok {
		return s
	}
	return ""
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
