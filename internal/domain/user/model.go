package user

import (
	"time"

	"github.com/reinsman/reinhub/internal/dbs"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	OAuthID   string    `gorm:"unique;size:64;column:oauth_id" json:"oauth_id"`
	AccountID string    `gorm:"size:16" json:"account_id"`
	UserName  string    `gorm:"size:32" json:"user_name"`
	Email     string    `gorm:"size:64" json:"email"`
}

func init() {
	dbs.RegisterModels(&User{})
}
