package rsp

import (
	"fmt"
	"time"

	"github.com/reinsman/reinhub/pkg/api/errs"
)

type BaseRsp struct {
	Code int    `json:"Code"`
	Msg  string `json:"Msg"`
	Cost int64  `json:"_cost,omitempty"`
	Err  string `json:"_err,omitempty"`
}

func (rsp *BaseRsp) String() string {
	return fmt.Sprintf("code:%d, msg:%s, cost:%d, error:%s", rsp.Code, rsp.Msg, rsp.Cost, rsp.Err)
}

// data 为结构体或结构体指针
type Rsp struct {
	BaseRsp
	Data any `json:"Data,omitempty"`
}

type Empty struct{}
type Ignore struct{}

func (rsp *BaseRsp) Set(fields ...interface{}) {
	if len(fields) > 3 {
		panic("basersp set() msg invalid")
	} else {
		for _, argv := range fields {
			switch v := argv.(type) {
			case errs.Code:
				rsp.Code = int(v)
				rsp.Msg = v.Msg()
			case string:
				rsp.Err = v
			case error:
				rsp.Err = v.Error()
			case time.Duration:
				rsp.Cost = v.Milliseconds()
			case time.Time:
				rsp.Cost = time.Since(v).Milliseconds()
			default:
				panic("no support type")
			}
		}
	}
}
func (rsp *BaseRsp) Since(beg time.Time) {
	rsp.Cost = time.Since(beg).Milliseconds()
}
func (rsp *BaseRsp) Spend(dura time.Duration) {
	rsp.Cost = dura.Milliseconds()
}

func (rsp *BaseRsp) Succ(argvs ...interface{}) {
	argvs = append(argvs, errs.SUCC)
	rsp.Set(argvs...)
}

func (rsp *BaseRsp) IsSucc() bool {
	return rsp.Code == int(errs.SUCC)
}

func (rsp *BaseRsp) Fail(argvs ...interface{}) {
	argvs = append(argvs, errs.FAIL)
	rsp.Set(argvs...)
}

func (rsp *BaseRsp) Pack(beg time.Time, err error, lang ...string) {
	rsp.Since(beg)
	if err == nil {
		rsp.Succ()
	} else {
		if ce, ok := err.(*errs.CodeErr); ok {
			rsp.Set(ce.C, ce.Error())
		} else {
			rsp.Fail(err)
		}
	}
}

func (rsp *BaseRsp) ErrParas(argvs ...interface{}) {
	argvs = append(argvs, errs.BAD_PARAS)
	rsp.Set(argvs...)
}

func (rsp *BaseRsp) ErrAuth(argvs ...interface{}) {
	argvs = append(argvs, errs.AUTH_FAIL)
	rsp.Set(argvs...)
}

func (rsp *BaseRsp) ErrDb(argvs ...interface{}) {
	argvs = append(argvs, errs.DB_FAIL)
	rsp.Set(argvs...)
}

func (rsp *BaseRsp) ErrNet(argvs ...interface{}) {
	argvs = append(argvs, errs.NET_ERR)
	rsp.Set(argvs...)
}
func (rsp *BaseRsp) ErrLimited(argvs ...interface{}) {
	argvs = append(argvs, errs.RATE_LIMITED)
	rsp.Set(argvs...)
}
