package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-book/rest-api/example8-rest-api/handler"
	"github.com/go-book/rest-api/example8-rest-api/middleware"
	"github.com/go-book/rest-api/example8-rest-api/store"
)

func newRouter(userStore *store.MemoryStore) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logger())

	userHandler := handler.NewUserHandler(userStore)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.POST("", userHandler.CreateUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.PATCH("/:id", userHandler.PatchUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	return r
}
