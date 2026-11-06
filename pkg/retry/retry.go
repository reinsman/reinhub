package retry

import (
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"time"

	log "log/slog"

	"github.com/reinsman/reinhub/pkg/httpx"
)

type retryErr struct {
	desc   string
	origin error
}

func NewRetryErr(desc string) error {
	return &retryErr{desc: desc}
}
func (e *retryErr) Error() string {
	return e.desc
}

// 如果是，同时返回源错误
func IsRetryErr(err error) (error, bool) {
	if re, ok := err.(*retryErr); ok {
		return re.origin, true
	}

	return nil, false
}
func numberToInt(v any) int {
	switch v := v.(type) {
	case int:
		return v
	case uint:
		return int(v)
	case int8:
		return int(v)
	case uint8:
		return int(v)
	case int16:
		return int(v)
	case uint16:
		return int(v)
	case int32:
		return int(v)
	case uint32:
		return int(v)
	case int64:
		return int(v)
	case uint64:
		return int(v)
	default:
		panic("unsupported type")
	}
}

// 获取调用者所在代码页和行号和函数名
func callerLine(stack_lv ...int) string {
	skip := 2
	if len(stack_lv) > 0 {
		skip = stack_lv[0]
	}
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	file = filepath.Base(file)
	funcName := filepath.Base(runtime.FuncForPC(pc).Name())
	return fmt.Sprintf("%s:%d-%s", file, line, funcName)
}

// Retry 会在出现网络错误时重试

type RetryLogBool bool      // 是否打印重试日志
type RetryDynSleepBool bool // 是否动态调整sleep时间
type RetryTagString string  // 要打印的tag
type RetryStackSkipInt int  // 打印堆栈的层级， 默认为2. 在Retry外多封装一层则需要加1

func Retry(f func() error, paras ...any) (err error) {
	retry_times := 0
	sleep := time.Second
	dynamic_sleep := RetryDynSleepBool(false)
	print_retry_log := RetryLogBool(false)
	log_tag := RetryTagString("")
	stack_lv := RetryStackSkipInt(2)

	for _, p := range paras {
		switch v := p.(type) {
		case int8, uint8, int16, uint16, int, uint, int32, uint32, int64, uint64:
			retry_times = numberToInt(v)
		case time.Duration:
			sleep = v
		case RetryDynSleepBool:
			dynamic_sleep = v
		case RetryLogBool:
			print_retry_log = v
		case RetryTagString:
			log_tag = v
		case RetryStackSkipInt:
			stack_lv = v
		default:
			panic("no support type")
		}
	}
	// 负数表示永远重试，0表示不重试
	if retry_times < 0 {
		retry_times = math.MaxInt
	}
	if print_retry_log && log_tag == "" {
		log_tag = RetryTagString(callerLine(int(stack_lv)))
	}
	tried := -1
	for {
		err = f()

		tried += 1
		if tried > 0 && print_retry_log {
			errmsg := ""
			if err != nil {
				errmsg = err.Error()
			}
			log.Warn(fmt.Sprintf("%s retried %d times, err: %s", log_tag, tried, errmsg))
		}

		if err == nil {
			return nil
		} else {
			if !httpx.IsNetErr(err) {
				return
			}
		}

		if tried >= retry_times {
			break
		}

		time.Sleep(sleep)
		if dynamic_sleep {
			sleep *= 2
			if sleep >= time.Minute*2 {
				sleep = time.Second * 32
			}
		}
	}
	return &retryErr{
		desc:   fmt.Sprintf("after %d attempts, last error: %s", tried, err),
		origin: err,
	}
}

func RetryErr[T error](f func() error, _ T, paras ...any) (err error) {
	retry_times := 0
	sleep := time.Second
	dynamic_sleep := RetryDynSleepBool(false)
	print_retry_log := RetryLogBool(false)
	log_tag := RetryTagString("")
	stack_lv := RetryStackSkipInt(2)

	for _, p := range paras {
		switch v := p.(type) {
		case int8, uint8, int16, uint16, int, uint, int32, uint32, int64, uint64:
			retry_times = numberToInt(v)
		case time.Duration:
			sleep = v
		case RetryDynSleepBool:
			dynamic_sleep = v
		case RetryLogBool:
			print_retry_log = v
		case RetryTagString:
			log_tag = v
		case RetryStackSkipInt:
			stack_lv = v
		default:
			panic("no support type")
		}
	}
	// 负数表示永远重试，0表示不重试
	if retry_times < 0 {
		retry_times = math.MaxInt
	}
	if print_retry_log && log_tag == "" {
		log_tag = RetryTagString(callerLine(int(stack_lv)))
	}
	tried := -1
	for {
		err = f()

		tried += 1
		if tried > 0 && print_retry_log {
			errmsg := ""
			if err != nil {
				errmsg = err.Error()
			}
			log.Warn(fmt.Sprintf("%s retried %d times, err: %s", log_tag, tried, errmsg))
		}

		if err == nil {
			return nil
		} else {
			switch err.(type) {
			case T:
			default:
				return
			}
		}

		if tried >= retry_times {
			break
		}

		time.Sleep(sleep)
		if dynamic_sleep {
			sleep *= 2
			if sleep >= time.Minute*2 {
				sleep = time.Second * 32
			}
		}
	}
	return &retryErr{
		desc:   fmt.Sprintf("after %d attempts, last error: %s", tried, err),
		origin: err,
	}
}
