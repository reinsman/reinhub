package errs

import "strings"

// 错误码
// 该页面为较常见的通用性的错误码
type Code int

// 基础错误码， 小于100
const (
	SUCC    Code = iota // 成功
	FAIL                // 笼统的失败
	EXT_ERR             // 上游\外部调用返回了业务错误

)

// 通用错误码(大多数系统都会出现的错误码)， 小于1000
const (
	BAD_PARAS    Code = iota + 100 // 参数非法
	AUTH_FAIL                      // 鉴权失败
	DB_FAIL                        // 数据库出错
	NET_ERR                        // 网络错误
	RATE_LIMITED                   // 被限流
)

// 各个服务错误码应该独立错误码区间，且大于1000

// 语言编码
type LangIdx int

const (
	LANG_zhCN LangIdx = iota
	LANG_en
	LANG_MAX
)

func ParseLangStr(lang string) LangIdx {
	lang = strings.ToLower(lang)
	switch lang {
	case "zh-cn", "zhcn", "cn":
		return LANG_zhCN
	case "en":
		return LANG_en
	default:
		return LANG_zhCN
	}
}

func init() {
	codeMsg[SUCC] = [LANG_MAX]string{"成功", "succ"}
	codeMsg[FAIL] = [LANG_MAX]string{"内部错误", "fail"}
	codeMsg[EXT_ERR] = [LANG_MAX]string{"上游发生错误", "upstream return error"}

	codeMsg[BAD_PARAS] = [LANG_MAX]string{"输入不符要求，请检查参数", "bad parameters"}
	codeMsg[AUTH_FAIL] = [LANG_MAX]string{"认证失败", "auth fail"}
	codeMsg[DB_FAIL] = [LANG_MAX]string{"内部错误，数据库操作出错", "database error"}
	codeMsg[NET_ERR] = [LANG_MAX]string{"内部错误，网络出现问题", "network error"}
	codeMsg[RATE_LIMITED] = [LANG_MAX]string{"接口被限流", "rate limited"}
}
