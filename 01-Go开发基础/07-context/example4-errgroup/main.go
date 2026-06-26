package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

var errPermissionDenied = errors.New("permission denied")

// 模拟查询用户信息
func getUser(ctx context.Context, userID int) (string, error) {
	select {
	case <-time.After(2 * time.Second):
		return fmt.Sprintf("user-%d", userID), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// 模拟查询权限
func getPermission(ctx context.Context, userID int) (string, error) {
	select {
	case <-time.After(1 * time.Second):
		return "", errPermissionDenied
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// 模拟查询菜单
func getMenu(ctx context.Context, userID int) (string, error) {
	select {
	case <-time.After(10 * time.Second): // 故意设置很长
		return fmt.Sprintf("menu-%d", userID), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func main() {
	fmt.Println("并发查询用户信息、权限、菜单...")

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	userID := 123

	var user, perm, menu string

	// 并发查询用户
	g.Go(func() error {
		var err error
		user, err = getUser(ctx, userID)
		if err != nil {
			fmt.Printf("查询用户失败: %v\n", err)
			return err
		}
		fmt.Printf("查询用户成功: %s\n", user)
		return nil
	})

	// 并发查询权限（会快速失败）
	g.Go(func() error {
		var err error
		perm, err = getPermission(ctx, userID)
		if err != nil {
			fmt.Printf("查询权限失败: %v\n", err)
			return err
		}
		fmt.Printf("查询权限成功: %s\n", perm)
		return nil
	})

	// 并发查询菜单（会被 errgroup 取消）
	g.Go(func() error {
		var err error
		menu, err = getMenu(ctx, userID)
		if err != nil {
			fmt.Printf("查询菜单失败: %v\n", err)
			return err
		}
		fmt.Printf("查询菜单成功: %s\n", menu)
		return nil
	})

	// 等待所有查询完成
	err := g.Wait()
	if err != nil {
		fmt.Printf("\n整体失败: %v\n", err)
		if errors.Is(err, errPermissionDenied) {
			fmt.Println("原因：权限不足")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("原因：超时")
		}
	} else {
		fmt.Printf("\n全部成功: user=%s, perm=%s, menu=%s\n", user, perm, menu)
	}

	fmt.Println("\n结论：任一任务返回错误后，errgroup 会取消仍在执行的任务")
}
