package service

import (
	"sync"
	"testing"

	"github.com/go-book/api-version/example5-admin-version/model"
)

func TestListReturnsCopies(t *testing.T) {
	service := NewUserService()

	users := service.List()
	users[0].Name = "changed"

	users = service.List()
	if users[0].Name == "changed" {
		t.Fatal("expected List to return copied users")
	}
}

func TestConcurrentCreateAndList(t *testing.T) {
	service := NewUserService()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			service.List()
		}()
		go func() {
			defer wg.Done()
			service.Create(&model.CreateUserInput{
				Name:  "alice",
				Email: "alice@example.com",
				Phone: "13800138000",
			})
		}()
	}
	wg.Wait()

	if got := len(service.List()); got != 22 {
		t.Fatalf("expected 22 users, got %d", got)
	}
}
