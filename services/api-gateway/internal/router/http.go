package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/services/api-gateway/internal/handler"
)

func Register(engine *gin.Engine, shopHandler *handler.ShopHandler) {
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
		})
	})

	api := engine.Group("/api")
	shop := api.Group("/shop")
	shop.POST("/register", shopHandler.Register)
	shop.POST("/login", shopHandler.Login)
}
