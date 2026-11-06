package app

import (
	"net/http"

	_ "github.com/reinsman/reinhub/docs"
	"github.com/reinsman/reinhub/internal/app/middleware"
	"github.com/reinsman/reinhub/internal/config"
	domaincontract "github.com/reinsman/reinhub/internal/domain"
	"github.com/reinsman/reinhub/internal/domain/self"
	"github.com/reinsman/reinhub/internal/domain/user"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 编译期检查两个领域是否实现了对方所需的窄接口。
var (
	_ domaincontract.UserCounter = (*user.Domain)(nil)
	_ domaincontract.PingGetter  = (*self.Domain)(nil)
)

// newRouter 组装应用依赖并注册 HTTP 路由。
func newRouter(db *gorm.DB, debug bool, config *config.Config) http.Handler {
	if debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(middleware.Base()...)

	// 两个领域只依赖集中定义的窄接口，由应用层完成运行时双向注入。
	userDomain := user.InitDomain(db)
	selfDomain := self.InitDomain(userDomain)
	if err := userDomain.WithPingGetter(selfDomain); err != nil {
		panic("组装用户领域与 Self 领域的双向依赖失败: " + err.Error())
	}

	selfGroup := engine.Group("/self")
	selfDomain.RegisterRoutes(selfGroup, config)

	apiV1Group := engine.Group("/api/v1")
	apiV1Group.Use(
		middleware.Authn(),
		middleware.Authz(),
	)
	userDomain.RegisterRoutes(apiV1Group)

	return engine
}
