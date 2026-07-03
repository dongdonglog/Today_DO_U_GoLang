package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-book/gin-framework/example7-admin-api/handler"
	"github.com/go-book/gin-framework/example7-admin-api/middleware"
	"github.com/go-book/gin-framework/example7-admin-api/store"
)

func newRouter(userStore *store.MemoryStore) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Logger())

	userHandler := handler.NewUserHandler(userStore)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		users := api.Group("/users")
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.POST("", userHandler.CreateUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	return r
}
