package user

import (
	"context"

	"gorm.io/gorm"
)

// Repository 定义用户领域的数据访问接口
type Repository interface {
	Insert(ctx context.Context, oauthID, accountID, userName, email string) (uint, error)
	GetByOAuthID(ctx context.Context, oauthID string) (*User, error)
	GetByUID(ctx context.Context, uid uint) (*User, error)
	GetByName(ctx context.Context, name string) (*User, error)
}

// gormRepository 基于 GORM 的 Repository 实现
type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Insert(ctx context.Context, oauthID, accountID, userName, email string) (uint, error) {
	u := User{
		OAuthID:   oauthID,
		AccountID: accountID,
		UserName:  userName,
		Email:     email,
	}
	rst := r.db.WithContext(ctx).FirstOrCreate(&u, User{OAuthID: oauthID})
	return u.ID, rst.Error
}

func (r *gormRepository) GetByOAuthID(ctx context.Context, oauthID string) (*User, error) {
	var u User
	rst := r.db.WithContext(ctx).Where("oauth_id = ?", oauthID).First(&u)
	if rst.Error != nil {
		if rst.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, rst.Error
	}
	return &u, nil
}

func (r *gormRepository) GetByUID(ctx context.Context, uid uint) (*User, error) {
	var u User
	rst := r.db.WithContext(ctx).Where("id = ?", uid).First(&u)
	if rst.Error != nil {
		if rst.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, rst.Error
	}
	return &u, nil
}

func (r *gormRepository) GetByName(ctx context.Context, name string) (*User, error) {
	var u User
	rst := r.db.WithContext(ctx).Where("user_name = ?", name).First(&u)
	if rst.Error != nil {
		if rst.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, rst.Error
	}
	return &u, nil
}
