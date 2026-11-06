package errs

import "fmt"

var (
	codeMsg = map[Code][LANG_MAX]string{}
)

func (c Code) Msg(langarg ...any) string {
	l := LANG_zhCN
	if len(langarg) > 0 {
		switch v := langarg[0].(type) {
		case int:
			l = LangIdx(v)
		case string:
			l = ParseLangStr(v)
		case LangIdx:
			l = v
		default:
			panic(fmt.Sprintf("invalid language: %v", langarg[0]))
		}
		if l >= LANG_MAX {
			panic(fmt.Sprintf("invalid language: %d", l))
		}
	}
	if msg, ok := codeMsg[c]; ok {
		return msg[l]
	}
	return fmt.Sprintf("unregistered error code: %d", int(c))
}

func (c Code) Int() int {
	return int(c)
}

func Map(i int) string {
	c := Code(i)
	return c.Msg()
}
