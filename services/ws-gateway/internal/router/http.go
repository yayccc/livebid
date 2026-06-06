package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/services/ws-gateway/internal/handler"
)

func Register(engine *gin.Engine, wsHandler *handler.WebSocketHandler) {
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
		})
	})

	engine.GET("/ws/live", wsHandler.ServeLive)
}
