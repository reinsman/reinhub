package errs

import (
	"errors"
	"fmt"
	"reflect"
)

// 自定义错误码与错误详情
// C 错误码
// E 用于调试的错误详情
type CodeErr struct {
	C Code
	E error
}

func New(fields ...any) error {
	e := CodeErr{
		C: SUCC,
	}

	for _, f := range fields {
		if f == nil {
			continue
		}
		switch v := f.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			c := reflect.ValueOf(v).Convert(reflect.TypeOf(0)).Int()
			e.C = Code(c)
		case Code:
			e.C = v
		case *CodeErr:
			if v == nil {
				continue
			}
			return v
		case string:
			e.E = errors.New(v)
		case error:
			e.E = v
		default:
			panic("no support type")
		}
	}

	return &e
}

func (e *CodeErr) Error() string {
	if e.E != nil {
		return e.E.Error()
	}
	return ""
}

func (e *CodeErr) String() string {
	return fmt.Sprintf("code: %d msg: %s error: %s", e.C, e.C.Msg(), e.Error())
}

// 下面快捷函数，如果传递了新的错误码会覆盖函数名暗示的错误码
func ErrFail(err ...any) error {
	args := make([]any, 0, len(err)+1)
	args = append(args, FAIL)
	args = append(args, err...)
	return New(args...)
}

func ErrExt(err ...any) error {
	args := make([]any, 0, len(err)+1)
	args = append(args, EXT_ERR)
	args = append(args, err...)
	return New(args...)
}

func ErrParas(err ...any) error {
	args := make([]any, 0, len(err)+1)
	args = append(args, BAD_PARAS)
	args = append(args, err...)
	return New(args...)

}
func ErrDB(err ...any) error {
	args := make([]any, 0, len(err)+1)
	args = append(args, DB_FAIL)
	args = append(args, err...)
	return New(args...)

}

func ErrAuth(err ...any) error {
	args := make([]any, 0, len(err)+1)
	args = append(args, AUTH_FAIL)
	args = append(args, err...)
	return New(args...)

}
func ErrNet(err ...any) error {
	args := make([]any, 0, len(err)+1)
	args = append(args, NET_ERR)
	args = append(args, err...)
	return New(args...)

}
func ErrRate(err ...any) error {
	args := make([]any, 0, len(err)+1)
	args = append(args, RATE_LIMITED)
	args = append(args, err...)
	return New(args...)

}
