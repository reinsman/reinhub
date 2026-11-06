package self

import (
	. "github.com/reinsman/reinhub/internal/ginutil"
	"github.com/reinsman/reinhub/internal/globals"
	pingapi "github.com/reinsman/reinhub/pkg/api/v1/ping"

	"github.com/gin-gonic/gin"
)

// @Summary      服务健康检查
// @Description  检查服务是否可以正常响应
// @Tags         /self
// @Accept       json
// @Produce      json
// @Success      200  {object}  rsp.Rsp{data=pingapi.Pong} "成功code为0，其他code为失败,失败时data可能为nil"
// @Router       /self/ping [get]
func ping(c *gin.Context) {
	health, err := getPing(c.Request.Context())
	if err != nil {
		Done(c, err)
		return
	}
	data := pingapi.Pong{
		CommitId: globals.CommitId,
		Health:   health,
	}
	Done(c, data)
}
