package main

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/proto"

	modelpb "github.com/go-book/protobuf/proto/model"
)

// 演示 protobuf 的线格式：序列化成紧凑的二进制，并和 JSON 对比体积。
func main() {
	u := &modelpb.User{
		Id:       1001,
		Username: "alice",
		Email:    "alice@example.com",
		Status:   1,
	}

	// protobuf 二进制
	pbBytes, err := proto.Marshal(u)
	if err != nil {
		panic(err)
	}

	// 等价 JSON
	jsonBytes, _ := json.Marshal(map[string]any{
		"id":       1001,
		"username": "alice",
		"email":    "alice@example.com",
		"status":   1,
	})

	fmt.Printf("protobuf: %d 字节  %x\n", len(pbBytes), pbBytes)
	fmt.Printf("JSON:     %d 字节  %s\n", len(jsonBytes), jsonBytes)
	fmt.Printf("体积比:   protobuf 约为 JSON 的 %.0f%%\n",
		float64(len(pbBytes))/float64(len(jsonBytes))*100)

	// 反序列化
	var decoded modelpb.User
	if err := proto.Unmarshal(pbBytes, &decoded); err != nil {
		panic(err)
	}
	fmt.Printf("反序列化: id=%d username=%s\n", decoded.GetId(), decoded.GetUsername())

	// 逐字节解释前几个 tag：(field_number << 3) | wire_type
	// 0x08 = field 1, varint; 0x09 = 1001 的 varint 编码...
	fmt.Printf("\n第一个字节 0x%02x => field=%d wiretype=%d (varint)\n",
		pbBytes[0], pbBytes[0]>>3, pbBytes[0]&0x7)
}
