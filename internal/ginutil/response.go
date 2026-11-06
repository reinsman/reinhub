package ginutil

import (
	"net/http"
	"time"

	"github.com/reinsman/reinhub/pkg/api/errs"
	"github.com/reinsman/reinhub/pkg/api/rsp"
	applog "github.com/reinsman/reinhub/pkg/log"

	"github.com/gin-gonic/gin"
)

// Done 统一响应封装。
//
// Done(c, code, str)
// Done(c, code, err)
// Done(c, err)
// Done(c, code)
// Done(c, str)
// Done(c, data)
// Done(c, data, err)
// Done(c)
func Done(c *gin.Context, fields ...any) {
	if len(fields) > 2 {
		panic("Done() parameters invalid")
	}
	err := errs.New()
	rsp := rsp.Rsp{}

	if len(fields) == 1 {
		switch value := fields[0].(type) {
		case errs.Code:
			err = errs.New(value)
		case string:
			err = errs.ErrFail(value)
		case *errs.CodeErr:
			err = value
		case error:
			err = errs.ErrFail(value)
		case nil:
		default:
			rsp.Data = value
		}
	} else if len(fields) == 2 {
		if code, ok := fields[0].(errs.Code); ok {
			err = errs.New(code, fields[1])
		} else {
			rsp.Data = fields[0]
			err = errs.New(fields[1])
		}
	}

	//lang, _ := c.Cookie("lang")
	begin := time.Now()
	if requestLog, ok := applog.GetAccessLog(c.Request.Context()); ok {
		begin = requestLog.Begin
	}
	rsp.Pack(begin, err)
	applog.SetAccessResponse(c.Request.Context(), rsp.Code, rsp.Msg, rsp.Err)
	c.Set("code", rsp.Code)
	if gin.Mode() == gin.ReleaseMode {
		rsp.Err = ""
		rsp.Cost = 0
	}
	c.JSON(http.StatusOK, rsp)
}

// Todo 未实现接口的占位响应。
func Todo(c *gin.Context) {
	Done(c, errs.FAIL, "undone")
}
