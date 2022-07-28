package routers

import (
	"github.com/Crossbell-Box/OperatorSync/app/server/handlers/v1"
	"github.com/gin-gonic/gin"
)

func V1Endpoints(rg *gin.RouterGroup) {
	rg.GET("/:character", v1.CheckCharacter)
	rg.GET("/:character/account", v1.ListAccounts)
	rg.POST("/:character/account/bind/:platform/:username", v1.BindAccount)
	rg.DELETE("/:character/account/unbind/:platform/:username", v1.UnbindAccount)
	rg.GET("/:character/media", v1.ListMedias)
	//rg.GET("/:character/feed") // Get feeds (not enabled)

	rg.GET("/feed/:platform/:username", v1.ListSingleAccountFeeds)
}