package middleware

import (
	"context"
	"net/http"

	"github.com/reinsman/reinhub/pkg/api/rsp"
	applog "github.com/reinsman/reinhub/pkg/log"

	"github.com/gin-gonic/gin"
)

const trustedHeaderName = "_HCID"

// Authn 验证网关是否已经完成请求身份认证。
func Authn() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader(trustedHeaderName) == "" {
			response := rsp.BaseRsp{}
			response.ErrAuth("trusted header not found")
			abortWithBusinessError(c, response)
			return
		}
		c.Next()
	}
}

// Authz 验证已认证的调用方是否有权访问请求资源。
func Authz() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requestAuthorized(c.Request.Context(), c.Request) {
			response := rsp.BaseRsp{}
			response.ErrAuth("permission denied")
			abortWithBusinessError(c, response)
			return
		}
		c.Next()
	}
}

// requestAuthorized 是调用远程鉴权服务的占位实现。
// 在接入实际鉴权服务前，当前实现允许所有已认证请求通过。
func requestAuthorized(_ context.Context, _ *http.Request) bool {
	return true
}

func abortWithBusinessError(c *gin.Context, response rsp.BaseRsp) {
	applog.SetAccessResponse(c.Request.Context(), response.Code, response.Msg, response.Err)
	c.Set("code", response.Code)
	c.AbortWithStatusJSON(http.StatusOK, response)
}
