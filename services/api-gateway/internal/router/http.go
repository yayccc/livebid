package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/services/api-gateway/internal/handler"
	"github.com/yayccc/livebid/services/api-gateway/internal/middleware"
)

func Register(engine *gin.Engine, shopHandler *handler.ShopHandler, userHandler *handler.UserHandler, goodsHandler *handler.GoodsHandler, fileHandler *handler.FileHandler, liveHandler *handler.LiveHandler, auctionHandler *handler.AuctionHandler, shopJWTManager *auth.JWTManager, userJWTManager *auth.JWTManager) {
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

	users := api.Group("/users")
	users.POST("/register", userHandler.Register)
	users.POST("/login", userHandler.Login)
	users.GET("/:id", userHandler.Get)

	goods := api.Group("/goods")
	goods.POST("/batch", goodsHandler.BatchGetGoods)
	goods.GET("", goodsHandler.List)
	goods.GET("/:id", goodsHandler.Get)

	files := api.Group("/files")
	files.POST("/upload", fileHandler.Upload)

	// 需要商户认证的接口，使用RequireShopAuth中间件进行保护
	merchant := api.Group("")
	merchant.Use(middleware.RequireShopAuth(shopJWTManager))

	merchantShop := merchant.Group("/shop")
	merchantShop.GET("/me", shopHandler.GetCurrent)
	merchantShop.PUT("/:id", shopHandler.Update)

	merchantGoods := merchant.Group("/goods")
	merchantGoods.GET("/shop/list", goodsHandler.ListShopGoods)
	merchantGoods.POST("", goodsHandler.Create)
	merchantGoods.PUT("/:id", goodsHandler.Update)
	merchantGoods.DELETE("/:id", goodsHandler.Delete)
	merchantGoods.PUT("/:id/on-sale", goodsHandler.PutOnSale)
	merchantGoods.PUT("/:id/off-sale", goodsHandler.PutOffSale)

	user := api.Group("")
	user.Use(middleware.RequireUserAuth(userJWTManager))

	userUsers := user.Group("/users")
	userUsers.POST("/avatar/upload", fileHandler.Upload)
	userUsers.POST("/address", userHandler.CreateAddress)
	userUsers.GET("/address/list", userHandler.ListAddresses)
	userUsers.GET("/address/:id", userHandler.GetAddress)
	userUsers.PUT("/address/:id", userHandler.UpdateAddress)
	userUsers.DELETE("/address/:id", userHandler.DeleteAddress)
	userUsers.PUT("/address/:id/default", userHandler.SetDefaultAddress)
	userUsers.PUT("/:id", userHandler.UpdateCurrent)

	live := api.Group("/live")
	live.GET("/rooms", liveHandler.ListLiveRooms)
	live.GET("/rooms/:id", liveHandler.GetLiveRoom)

	auction := api.Group("/auctions")
	//TODO: 接口设计有问题
	auction.GET("", auctionHandler.ListShop)
	auction.GET("/by-goods/:goods_id", auctionHandler.GetByGoods)
	auction.GET("/:id", auctionHandler.Get)
	auction.GET("/:id/bids", auctionHandler.ListBidRecords)

	merchantAuction := merchant.Group("/auctions")
	merchantAuction.POST("", auctionHandler.Create)
	merchantAuction.PUT("/:id", auctionHandler.Update)
	merchantAuction.POST("/:id/start", auctionHandler.Start)
	merchantAuction.POST("/:id/finish", auctionHandler.Finish)
	merchantAuction.POST("/:id/cancel", auctionHandler.Cancel)
	merchantAuction.DELETE("/:id", auctionHandler.Delete)

	userAuction := user.Group("/auctions")
	userAuction.POST("/:id/bids", auctionHandler.PlaceBid)

	merchantLive := merchant.Group("/live")
	merchantLive.POST("/rooms", liveHandler.CreateLiveRoom)
	merchantLive.POST("/rooms/:id/start", liveHandler.StartLive)
	merchantLive.POST("/rooms/:id/end", liveHandler.EndLive)
	merchantLive.GET("/rooms/:id/stream", liveHandler.GetLiveStreamInfo)

	srs := api.Group("/srs")
	srs.POST("/callbacks/publish", liveHandler.HandleSRSPublishCallback)
	srs.POST("/callbacks/unpublish", liveHandler.HandleSRSUnpublishCallback)
}
