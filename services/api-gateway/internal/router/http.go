package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/services/api-gateway/internal/handler"
)

func Register(engine *gin.Engine, shopHandler *handler.ShopHandler, goodsHandler *handler.GoodsHandler, liveHandler *handler.LiveHandler) {
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

	goods := api.Group("/goods")
	goods.POST("/cover/upload", goodsHandler.UploadCover)
	goods.GET("/shop/list", goodsHandler.ListShopGoods)
	goods.POST("/batch", goodsHandler.BatchGetGoods)
	goods.POST("", goodsHandler.Create)
	goods.GET("", goodsHandler.List)
	goods.GET("/:id", goodsHandler.Get)
	goods.PUT("/:id", goodsHandler.Update)
	goods.DELETE("/:id", goodsHandler.Delete)
	goods.PUT("/:id/on-sale", goodsHandler.PutOnSale)
	goods.PUT("/:id/off-sale", goodsHandler.PutOffSale)

	live := api.Group("/live")
	live.POST("/rooms", liveHandler.CreateLiveRoom)
	live.GET("/rooms", liveHandler.ListLiveRooms)
	live.GET("/rooms/:id", liveHandler.GetLiveRoom)
	live.POST("/rooms/:id/start", liveHandler.StartLive)
	live.POST("/rooms/:id/end", liveHandler.EndLive)
	live.GET("/rooms/:id/stream", liveHandler.GetLiveStreamInfo)

	srs := api.Group("/srs")
	srs.POST("/callbacks/publish", liveHandler.HandleSRSPublishCallback)
	srs.POST("/callbacks/unpublish", liveHandler.HandleSRSUnpublishCallback)
}
