package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/services/api-gateway/internal/identity"
)

func currentShopID(c *gin.Context) (int64, bool) {
	shopID, ok := identity.ShopID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "missing auth")
		return 0, false
	}
	return shopID, true
}
