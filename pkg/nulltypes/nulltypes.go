package nulltypes

import (
	"database/sql"
	"reflect"
)

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func Nullstr(t *string) sql.NullString {
	if t == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{Valid: true, String: *t}
}
func Nullint[T integer](t *T) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Valid: true, Int64: int64(*t)}
}

func Nilstr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func Nilint[T integer](s sql.NullInt64, opt ...T) *T {
	if !s.Valid {
		return nil
	}
	r := T(s.Int64)
	return &r
}

func Tonull[T any, R any](t *T, _r R) R {
	rtyp := reflect.TypeOf(_r)

	valkind := rtyp.Field(0).Type.Kind()
	switch valkind {
	case reflect.Bool:
		if t == nil {
			return any(sql.NullBool{Valid: false}).(R)
		} else {
			return any(sql.NullBool{Valid: true, Bool: reflect.ValueOf(*t).Bool()}).(R)
		}
	case reflect.Int64:
		if t == nil {
			return any(sql.NullInt64{Valid: false}).(R)
		} else {

			return any(sql.NullInt64{Valid: true, Int64: convint(*t, int64(0))}).(R)
		}
	case reflect.Int32:
		if t == nil {
			return any(sql.NullInt32{Valid: false}).(R)
		} else {

			return any(sql.NullInt32{Valid: true, Int32: convint(*t, int32(0))}).(R)
		}
	case reflect.Int16:
		if t == nil {
			return any(sql.NullInt16{Valid: false}).(R)
		} else {

			return any(sql.NullInt16{Valid: true, Int16: convint(*t, int16(0))}).(R)
		}
	case reflect.Int8, reflect.Uint8:
		if t == nil {
			return any(sql.NullByte{Valid: false}).(R)
		} else {

			return any(sql.NullByte{Valid: true, Byte: convint(*t, byte(0))}).(R)
		}
	case reflect.String:
		if t == nil {
			return any(sql.NullString{Valid: false}).(R)
		} else {
			return any(sql.NullString{Valid: true, String: any(*t).(string)}).(R)
		}
	case reflect.Float64:
		if t == nil {
			return any(sql.NullFloat64{Valid: false}).(R)
		} else {
			ek := reflect.TypeOf(t).Elem().Kind()
			if ek != reflect.Float64 && ek != reflect.Float32 {
				panic("unsupported float type")
			}
			if ek == reflect.Float32 {
				return any(sql.NullFloat64{Valid: true, Float64: float64(any(*t).(float32))}).(R)
			}
			return any(sql.NullFloat64{Valid: true, Float64: any(*t).(float64)}).(R)
		}
	}
	return _r
}

func convint[R integer](value any, _ R) R {
	// 将输入值转换为 int64 类型
	switch ival := any(value).(type) {
	case int8:
		return R(int64(ival))
	case int16:
		return R(int64(ival))
	case int32:
		return R(int64(ival))
	case int64:
		return R(int64(ival))
	case int:
		return R(int64(ival))
	case uint8:
		return R(int64(ival))
	case uint16:
		return R(int64(ival))
	case uint32:
		return R(int64(ival))
	case uint64:
		return R(int64(ival))
	case uint:
		return R(int64(ival))
	case uintptr:
		return R(int64(ival))
	default:
		panic("unsupported type")
	}
}
