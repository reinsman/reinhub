package nulltypes

import (
	"database/sql"
	"testing"
)

func TestTonull(t *testing.T) {
	bo := true
	var nilbool *bool = nil

	i := 1
	var nilint *int = nil

	i64 := int64(1)
	var nilint64 *int64 = nil

	by := byte(1)
	var nilbyte *byte = nil

	s := "1"
	var nilstring *string = nil

	f := 0.1
	var nilfloat *float64 = nil

	nullbool := Tonull(&bo, sql.NullBool{})
	if !nullbool.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullbool)
	nullbool = Tonull(nilbool, sql.NullBool{})
	if nullbool.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullbool)

	nullint := Tonull(&i, sql.NullInt64{})
	if !nullint.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullint)
	nullint = Tonull(nilint, sql.NullInt64{})
	if nullint.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullint)

	nullint64 := Tonull(&i64, sql.NullInt64{})
	if !nullint64.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullint64)
	nullint64 = Tonull(nilint64, sql.NullInt64{})
	if nullint64.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullint64)

	nullbyte := Tonull(&by, sql.NullByte{})
	if !nullbyte.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullbyte)
	nullbyte = Tonull(nilbyte, sql.NullByte{})
	if nullbyte.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullbyte)

	nullstring := Tonull(&s, sql.NullString{})
	if !nullstring.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullstring)
	nullstring = Tonull(nilstring, sql.NullString{})
	if nullstring.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullstring)

	nullfloat := Tonull(&f, sql.NullFloat64{})
	if !nullfloat.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullfloat)
	nullfloat = Tonull(nilfloat, sql.NullFloat64{})
	if nullfloat.Valid {
		t.Fatal("not valid")
	}
	t.Log(nullfloat)

	nulldiffint := Tonull(&i, sql.NullInt64{})
	if !nulldiffint.Valid {
		t.Fatal("not valid")
	}
	t.Log(nulldiffint)
	nulldiffint = Tonull(nilint, sql.NullInt64{})
	if nulldiffint.Valid {
		t.Fatal("not valid")
	}
	t.Log(nulldiffint)

	rpi := Nilint[int](sql.NullInt64{Int64: 12345, Valid: true})
	t.Log(*rpi)
	rpi = Nilint[int](sql.NullInt64{Int64: 12345, Valid: false})
	t.Log(rpi == nil)
	rpi = Nilint(sql.NullInt64{Int64: 12345, Valid: false}, int(12345))
	t.Log(rpi == nil)
}
