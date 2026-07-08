package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 原始密码
	password := "123456"
	fmt.Printf("原始密码: %s\n", password)

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("加密后: %s\n", string(hashedPassword))

	// 验证正确密码
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		fmt.Println("密码验证失败")
	} else {
		fmt.Println("密码验证成功")
	}

	// 验证错误密码
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte("wrongpassword"))
	if err != nil {
		fmt.Println("错误密码验证失败（预期行为）")
	} else {
		fmt.Println("错误密码验证成功（不应该发生）")
	}

	// 多次加密同一密码，结果不同（因为盐值不同）
	hashedPassword2, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	fmt.Printf("\n第一次加密: %s\n", string(hashedPassword))
	fmt.Printf("第二次加密: %s\n", string(hashedPassword2))
	fmt.Printf("结果相同吗? %v\n", string(hashedPassword) == string(hashedPassword2))

	// 但都能验证成功
	err1 := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	err2 := bcrypt.CompareHashAndPassword(hashedPassword2, []byte(password))
	fmt.Printf("第一次能验证成功? %v\n", err1 == nil)
	fmt.Printf("第二次能验证成功? %v\n", err2 == nil)
}
