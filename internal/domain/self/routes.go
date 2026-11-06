package self

import (
	"github.com/gin-gonic/gin"
	"github.com/reinsman/reinhub/internal/config"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes 注册服务自省领域的 HTTP 路由。
func (d *Domain) RegisterRoutes(router gin.IRoutes, cfg *config.Config) {
	router.GET("/ping", ping)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

}
