package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/services/api-gateway/internal/handler"
	"github.com/yayccc/livebid/services/api-gateway/internal/middleware"
)

func Register(engine *gin.Engine, shopHandler *handler.ShopHandler, goodsHandler *handler.GoodsHandler, fileHandler *handler.FileHandler, liveHandler *handler.LiveHandler, auctionHandler *handler.AuctionHandler, shopJWTManager *auth.JWTManager, _ *auth.JWTManager) {
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
		})
	})

	// API路由分组
	api := engine.Group("/api")
	shop := api.Group("/shop")

	// 商户注册和登录接口不需要认证,直接注册在对应分组下
	shop.POST("/register", shopHandler.Register)
	shop.POST("/login", shopHandler.Login)

	goods := api.Group("/goods")
	goods.POST("/batch", goodsHandler.BatchGetGoods)
	goods.GET("", goodsHandler.List)
	goods.GET("/:id", goodsHandler.Get)

	files := api.Group("/files")
	files.POST("/upload", fileHandler.Upload)

	auctions := api.Group("/auctions")
	auctions.GET("/goods/:goods_id", auctionHandler.GetByGoods)
	auctions.GET("/:id/bids", auctionHandler.ListBidRecords)

	// 需要商户认证的接口，使用RequireShopAuth中间件进行保护
	merchant := api.Group("")
	merchant.Use(middleware.RequireShopAuth(shopJWTManager))

	merchantShop := merchant.Group("/shop")
	merchantShop.GET("/me", shopHandler.GetCurrent)
	merchantShop.PUT("/:id", shopHandler.Update)

	merchantGoods := merchant.Group("/goods")
	merchantGoods.POST("/cover/upload", fileHandler.UploadGoodsCover)
	merchantGoods.GET("/shop/list", goodsHandler.ListShopGoods)
	merchantGoods.POST("", goodsHandler.Create)
	merchantGoods.PUT("/:id", goodsHandler.Update)
	merchantGoods.DELETE("/:id", goodsHandler.Delete)
	merchantGoods.PUT("/:id/on-sale", goodsHandler.PutOnSale)
	merchantGoods.PUT("/:id/off-sale", goodsHandler.PutOffSale)

	merchantAuctions := merchant.Group("/auctions")
	merchantAuctions.GET("/shop", auctionHandler.ListShop)
	merchantAuctions.POST("", auctionHandler.Create)
	merchantAuctions.PUT("/:id", auctionHandler.Update)
	merchantAuctions.POST("/:id/start", auctionHandler.Start)
	merchantAuctions.POST("/:id/finish", auctionHandler.Finish)
	merchantAuctions.POST("/:id/cancel", auctionHandler.Cancel)
	merchantAuctions.DELETE("/:id", auctionHandler.Delete)

	auctions.GET("/:id", auctionHandler.Get)

	live := api.Group("/live")
	live.GET("/rooms", liveHandler.ListLiveRooms)
	live.GET("/rooms/:id", liveHandler.GetLiveRoom)

	merchantLive := merchant.Group("/live")
	merchantLive.POST("/rooms", liveHandler.CreateLiveRoom)
	merchantLive.POST("/rooms/:id/start", liveHandler.StartLive)
	merchantLive.POST("/rooms/:id/end", liveHandler.EndLive)
	merchantLive.GET("/rooms/:id/stream", liveHandler.GetLiveStreamInfo)

	srs := api.Group("/srs")
	srs.POST("/callbacks/publish", liveHandler.HandleSRSPublishCallback)
	srs.POST("/callbacks/unpublish", liveHandler.HandleSRSUnpublishCallback)
}
