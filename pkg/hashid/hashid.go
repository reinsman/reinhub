package hashid

import (
	"os"

	"github.com/speps/go-hashids/v2"
)

var hashids_salt = "whyflowerssored"

func init() {
	if val, ok := os.LookupEnv("e_hashids_salt"); ok {
		hashids_salt = val
	}
}

func HashIdSalt(salt string) {
	hashids_salt = salt
}

func EnHashId(num int64, argvs ...any) string {
	hd := hashids.NewData()
	hd.Salt = hashids_salt
	hd.MinLength = 8
	for _, argv := range argvs {
		switch v := argv.(type) {
		case string:
			hd.Alphabet = v
		case int:
			hd.MinLength = v
		default:
			panic("unknown type")
		}
	}
	h, _ := hashids.NewWithData(hd)
	if r, err := h.EncodeInt64([]int64{num}); err != nil {
		panic(err)
	} else {
		return r
	}
}
func DeHashId(id string, argvs ...any) (int64, error) {
	hd := hashids.NewData()
	hd.Salt = hashids_salt
	hd.MinLength = 8
	for _, argv := range argvs {
		switch v := argv.(type) {
		case string:
			hd.Alphabet = v
		case int:
			hd.MinLength = v
		default:
			panic("unknown type")
		}
	}
	h, _ := hashids.NewWithData(hd)
	if r, err := h.DecodeInt64WithError(id); err != nil {
		return 0, err
	} else {
		return r[0], nil
	}
}
