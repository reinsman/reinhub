package dbs

import (
	"errors"

	"gorm.io/gorm"
)

func IsNotFoundErr(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
func IsDupErr(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
