package user

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册用户领域的 HTTP 路由。
func (d *Domain) RegisterRoutes(router gin.IRoutes) {
	router.PUT("/dummy/:id", dummy)
	router.GET("/user", userInfo)
}
