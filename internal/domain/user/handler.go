package user

import (
	. "github.com/reinsman/reinhub/internal/ginutil"
	"github.com/reinsman/reinhub/pkg/api/errs"
	userapi "github.com/reinsman/reinhub/pkg/api/v1/user"
	"github.com/reinsman/reinhub/pkg/strutil"

	log "log/slog"

	"github.com/gin-gonic/gin"
)

// UserInfo 获取用户信息
//
//	@Summary		获取用户信息
//	@Description	按 uid 或 name 查询用户，都不传则查当前登录用户
//	@Tags			/api/v1
//	@Produce		json
//	@Param			uid		query		uint	false	"用户ID"
//	@Param			name	query		string	false	"用户名"
//	@Success		200		{object}	rsp.Rsp{data=userapi.UserInfoData}
//	@Router			/api/v1/user [get]
func userInfo(c *gin.Context) {
	var err error

	req := userapi.UserInfoReq{}

	if err = c.ShouldBindQuery(&req); err != nil {
		Done(c, errs.ErrParas(err))
		return
	}
	ctx := c.Request.Context()
	var data *userapi.UserInfoData
	if req.Name != "" {
		data, err = getByName(ctx, req.Name)
	} else if req.UID != 0 {
		data, err = getByUID(ctx, req.UID)
	}
	Done(c, data, err)
}

// Dummy 演示参数绑定。
//
//	@Summary		参数绑定示例
//	@Description	演示参数绑定
//	@Tags			/api/v1
//	@Accept			json
//	@Produce		json
//	@Param			_HCID	header		int					true	"用户id"
//	@Param			id		path		int					true	"id"
//	@Param			status	query		string				false	"用户状态"	Enums(active, inactive, pending)
//	@Param			offset	query		int					false	"分页偏移"	default(0)
//	@Param			limit	query		int					false	"每页数量"	default(10)
//	@Param			body	body		userapi.DummyReq	true	"请求体"
//	@Success		200		{object}	rsp.Rsp{data=userapi.DummyData}	"成功code为0，其他code为失败,失败时data可能为nil"
//	@Router			/api/v1/dummy/{id} [put]
func dummy(c *gin.Context) {
	var err error

	_, err = strutil.ParseStr(c.GetHeader("_HCID"), func(val int) bool { return val > 0 })
	if err != nil {
		Done(c, errs.ErrParas(err))
		return
	}
	_, err = strutil.ParseStr(c.Param("id"), func(val int) bool { return val > 0 })
	if err != nil {
		Done(c, errs.ErrParas())
		return
	}

	_, err = strutil.ParseStrOption(c.Query("offset"), 0, func(val int) bool { return val >= 0 })
	if err != nil {
		Done(c, errs.ErrParas(err))
		return
	}
	_, err = strutil.ParseStrOption(c.Query("limit"), 10, func(val int) bool { return val > 0 })
	if err != nil {
		Done(c, errs.ErrParas(err))
		return
	}
	_, err = strutil.ParseStrOption(c.Query("rgid"), nil, func(val *int) bool { return val != nil && *val >= 0 })
	_ = err
	_, err = strutil.ParseStrOption(c.Query("status"), "", func(val string) bool {
		return val == "active" || val == "inactive" || val == "pending"
	})
	_ = err

	req := userapi.DummyReq{}
	if err = c.ShouldBindJSON(&req); err != nil {
		Done(c, errs.ErrParas(err))
		return
	}
	log.InfoContext(c.Request.Context(), "hi dummy", "id", req.Id)
	if req.Panic {
		panic("dummy panic")
	}
	data, err := doDummy(c.Request.Context(), req.Id)
	Done(c, data, err)
}
