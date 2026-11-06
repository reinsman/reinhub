package errs

// 本服务的错误码
const (
	_ Code = iota + 2_000_000
	ALREADY_LOGIN
)

func init() {
	codeMsg[ALREADY_LOGIN] = [2]string{"已经是登录状态", "Already login"}
}
