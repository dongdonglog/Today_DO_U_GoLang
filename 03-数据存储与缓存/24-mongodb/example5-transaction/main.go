// example5-transaction 演示 MongoDB 多文档事务。
//
// 注意：MongoDB 事务需要副本集（replica set）。启动命令：
//
//	docker run --name go-book-mongo -p 27017:27017 -d mongo:7 --replSet rs0
//	docker exec go-book-mongo mongosh --quiet --eval "rs.initiate({_id:'rs0',members:[{_id:0,host:'localhost:27017'}]})"
//
// 等几秒让副本集选出 primary 后再跑本示例。前 4 个 example 不需要副本集。
//
// 场景：扣减库存 + 创建订单，必须原子——不能出现库存扣了但订单没创建，或反之。
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
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Product struct {
	ID    bson.ObjectID `bson:"_id,omitempty"`
	SKU   string        `bson:"sku"`
	Name  string        `bson:"name"`
	Stock int           `bson:"stock"`
	Price int64         `bson:"price"`
}

type Order struct {
	ID       bson.ObjectID `bson:"_id,omitempty"`
	OrderNo  string        `bson:"order_no"`
	UserID   int64         `bson:"user_id"`
	SKU      string        `bson:"sku"`
	Qty      int           `bson:"qty"`
	Amount   int64         `bson:"amount"`
	Status   string        `bson:"status"`
	CreateAt time.Time     `bson:"create_at"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(getenv("MONGO_URI", "mongodb://localhost:27017/?replicaSet=rs0")))
	if err != nil {
		log.Fatalf("连接失败（需副本集，见文件头注释）: %v", err)
	}
	defer client.Disconnect(ctx)
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("Ping primary 失败: %v\n请确认 mongo 已以副本集启动并 rs.initiate()", err)
	}
	fmt.Println("MongoDB 副本集连接成功")

	db := client.Database("shop_tx")
	products := db.Collection("products")
	orders := db.Collection("orders")
	products.Drop(ctx)
	orders.Drop(ctx)

	// 写入初始商品：库存 3
	products.InsertOne(ctx, Product{SKU: "KB-001", Name: "机械键盘", Stock: 3, Price: 25900})

	// ---- 成功事务：扣库存 + 建订单 ----
	fmt.Println("\n=== 成功事务：扣库存+建订单 ===")
	err = createOrderInTxn(ctx, client, products, orders, "KB-001", 2, 10086)
	fmt.Printf("结果: err=%v\n", err)

	var p Product
	products.FindOne(ctx, bson.M{"sku": "KB-001"}).Decode(&p)
	fmt.Printf("事务后库存: %d\n", p.Stock)
	count, _ := orders.CountDocuments(ctx, bson.D{})
	fmt.Printf("订单数: %d\n", count)

	// ---- 失败事务：库存不足回滚 ----
	fmt.Println("\n=== 失败事务：库存不足，全部回滚 ===")
	// 库存只剩 1，买 5 个应该失败回滚
	err = createOrderInTxn(ctx, client, products, orders, "KB-001", 5, 10087)
	fmt.Printf("结果: err=%v\n", err)

	products.FindOne(ctx, bson.M{"sku": "KB-001"}).Decode(&p)
	fmt.Printf("回滚后库存: %d（应仍为 1，没被扣成负数）\n", p.Stock)
	count2, _ := orders.CountDocuments(ctx, bson.D{})
	fmt.Printf("订单数: %d（仍为 1，没产生废订单）\n", count2)

	// ---- 模拟中间失败回滚 ----
	fmt.Println("\n=== 业务返回错误触发回滚 ===")
	err = createOrderWithError(ctx, client, products, orders, "KB-001", 1, 10088)
	fmt.Printf("结果: err=%v\n", err)
	products.FindOne(ctx, bson.M{"sku": "KB-001"}).Decode(&p)
	fmt.Printf("回滚后库存: %d（应仍为 1）\n", p.Stock)
}

func createOrderInTxn(ctx context.Context, client *mongo.Client, products, orders *mongo.Collection, sku string, qty int, user int64) error {
	sess, err := client.StartSession()
	if err != nil {
		return err
	}
	defer sess.EndSession(ctx)

	_, err = sess.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		// 1. 在事务会话里读
		var p Product
		if err := products.FindOne(sc, bson.M{"sku": sku}).Decode(&p); err != nil {
			return nil, err
		}
		if p.Stock < qty {
			// 返回 error 会触发 WithTransaction 回滚
			return nil, fmt.Errorf("库存不足: stock=%d need=%d", p.Stock, qty)
		}
		// 2. 扣库存（$inc 原子扣减）
		_, err := products.UpdateOne(sc,
			bson.M{"sku": sku, "stock": bson.M{"$gte": qty}}, // 加条件防并发
			bson.M{"$inc": bson.M{"stock": -qty}},
		)
		if err != nil {
			return nil, err
		}
		// 3. 建订单
		_, err = orders.InsertOne(sc, Order{
			OrderNo:  fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
			UserID:   user,
			SKU:      sku,
			Qty:      qty,
			Amount:   p.Price * int64(qty),
			Status:   "created",
			CreateAt: time.Now(),
		})
		return nil, err
	})
	return err
}

func createOrderWithError(ctx context.Context, client *mongo.Client, products, orders *mongo.Collection, sku string, qty int, user int64) error {
	sess, err := client.StartSession()
	if err != nil {
		return err
	}
	defer sess.EndSession(ctx)

	_, err = sess.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		// 先扣库存
		products.UpdateOne(sc, bson.M{"sku": sku}, bson.M{"$inc": bson.M{"stock": -qty}})
		// 模拟业务中途出错（下游 RPC 超时、校验失败等），返回 error 触发回滚
		return nil, fmt.Errorf("模拟业务校验失败")
	})
	return err
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
