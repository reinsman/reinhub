package strutil

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"time"

	"github.com/google/uuid"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyz1234567890"
)

func Md5str(origin string) string {
	h := md5.New()
	io.WriteString(h, origin)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Randn(n int) int {
	return rand.Intn(n)
}

func Randstr(n int, prefix ...string) string {
	if len(prefix) > 0 {
		n -= len(prefix[0])
	}
	b := make([]byte, n)
	for idx := 0; idx != n; idx++ {
		b[idx] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	ret := string(b)

	if len(prefix) > 0 && len(prefix[0]) > 0 {
		ret = prefix[0] + ret
	}
	return ret
}

func RandOf[T any](arr []T) T {
	return arr[rand.Intn(len(arr))]
}

func IsTestRun() bool {
	return flag.Lookup("test.v") != nil
}

func Uuid() string {
	return uuid.NewString()
}

func IsMapEqual[K, V comparable](m1, m2 map[K]V) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok || v1 != v2 {
			return false
		}
	}

	return true
}

func FmtShortTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("20060102150405")
}
func ParseShortTIme(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("20060102150405", s)
	if err != nil {
		return nil
	}
	return &t
}

type Parseable interface {
	string |
		bool |
		int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 | uintptr |
		float32 | float64 |
		*string |
		*bool |
		*int | *int8 | *int16 | *int32 | *int64 |
		*uint | *uint8 | *uint16 | *uint32 | *uint64 | *uintptr |
		*float32 | *float64
}

func ParseStr[T Parseable](val string, isValid ...func(T) bool) (T, error) {
	var result T

	if val == "" {
		return result, fmt.Errorf("value is empty")
	}

	if reflect.TypeOf(result).Kind() == reflect.Ptr {
		result = reflect.New(reflect.TypeOf(result).Elem()).Interface().(T)
		if _, err := fmt.Sscanf(val, "%v", result); err != nil {
			return result, err
		}
	} else {
		if _, err := fmt.Sscanf(val, "%v", &result); err != nil {
			return result, err
		}
	}

	if len(isValid) > 0 {
		if !isValid[0](result) {
			return result, fmt.Errorf("value is invalid")
		}
	}

	return result, nil
}

func ParseStrOption[T Parseable](val string, def T, isValid ...func(T) bool) (T, error) {
	if val == "" {
		return def, nil
	}

	return ParseStr(val, isValid...)
}

func MapKeys[T comparable, V any](collects map[T]V, exclude ...func(T) bool) []T {
	ret := make([]T, 0, len(collects))

	excl := func(T) bool { return false }
	if len(exclude) > 0 {
		excl = exclude[0]
	}
	for k := range collects {
		if !excl(k) {
			ret = append(ret, k)
		}
	}
	return ret
}
