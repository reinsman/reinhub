package middleware

import (
	applog "github.com/reinsman/reinhub/pkg/log"

	"github.com/gin-gonic/gin"
)

// 最基本、最先执行的中间件链，包含日志记录和 panic 恢复。
func Base() gin.HandlersChain {
	return gin.HandlersChain{
		applog.AccessLog(),
		gin.CustomRecovery(recoveryHandler),
	}
}
