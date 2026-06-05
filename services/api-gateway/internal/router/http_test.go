package router

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterAuctionRoutesDoesNotConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("register routes panicked: %v", recovered)
		}
	}()

	Register(gin.New(), nil, nil, nil, nil, nil, nil, nil)
}
