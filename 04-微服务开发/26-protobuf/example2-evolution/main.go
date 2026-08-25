package main

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	v1 "github.com/go-book/protobuf/proto/evolve/v1"
	v2 "github.com/go-book/protobuf/proto/evolve/v2"
)

// 演示 protobuf 的前向/后向兼容：
//   - 后向兼容：用 v2 客户端读 v1 产生的数据（新增字段缺失，读到零值）
//   - 前向兼容：用 v1 客户端读 v2 产生的数据（不认识的新字段被静默保留/忽略）
//
// 关键点：两个版本用不同的 Go 类型，但它们的线格式是兼容的，
// 因为字段编号一致、新字段用了新编号、废弃编号被 reserved。
func main() {
	// === 场景 A：v1 生产者 → v2 消费者（后向兼容）===
	v1Event := &v1.Event{
		Id:     "evt-1",
		Name:   "user.signup",
		Ts:     1728000000,
		Source: "web", // v2 里这个字段已经被废弃
	}
	wire, _ := proto.Marshal(v1Event)

	var v2Event v2.Event
	if err := proto.Unmarshal(wire, &v2Event); err != nil {
		panic(err)
	}
	fmt.Printf("A) v1->v2: id=%s name=%s channel=%q\n",
		v2Event.GetId(), v2Event.GetName(), v2Event.GetChannel())
	fmt.Println("   source 字段编号在 v2 被 reserved，数据被安全忽略，不会乱套到别的字段")

	// === 场景 B：v2 生产者 → v1 消费者（前向兼容）===
	v2Event2 := &v2.Event{
		Id:      "evt-2",
		Name:    "order.paid",
		Ts:      1728000100,
		Channel: "app", // v1 不认识这个字段
	}
	wire2, _ := proto.Marshal(v2Event2)

	var v1Event2 v1.Event
	if err := proto.Unmarshal(wire2, &v1Event2); err != nil {
		panic(err)
	}
	fmt.Printf("B) v2->v1: id=%s name=%s source=%q\n",
		v1Event2.GetId(), v1Event2.GetName(), v1Event2.GetSource())
	fmt.Println("   channel 是 v1 不认识的字段，被静默忽略，v1 仍能读到它认识的字段")

	// === 场景 C：v1 → v2 → v1 往返，未知字段被保留 ===
	// v1 数据经过 v2 反序列化再序列化后，重新序列化时未知字段仍会写回
	// （这是 protobuf Go 实现的默认行为）
	roundtrip, _ := proto.Marshal(&v2Event)
	var back v1.Event
	_ = proto.Unmarshal(roundtrip, &back)
	fmt.Printf("C) 往返: source 仍为 %q（未知字段保留）\n", back.GetSource())
}
