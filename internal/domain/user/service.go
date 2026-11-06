package user

import (
	"context"
	"errors"

	"github.com/reinsman/reinhub/pkg/api/errs"
	userapi "github.com/reinsman/reinhub/pkg/api/v1/user"
)

// getUserNum 返回用户数量。
// 当前作为跨领域调用示例，暂时直接返回固定值。
func getUserNum(_ context.Context) (int, error) {
	return 1, nil
}

// getByName 根据用户名查询用户。
func getByName(ctx context.Context, username string) (*userapi.UserInfoData, error) {
	ret := userapi.UserInfoData{}
	u, err := getRepo().GetByName(ctx, username)
	if err != nil {
		return nil, errs.ErrDB(err)
	}
	if u == nil {
		return nil, nil
	}
	ret.UID = u.ID
	ret.Name = u.UserName
	ret.AccountID = u.AccountID
	ret.Email = u.Email

	return &ret, nil
}

// getByUID 根据用户 ID 查询用户。
func getByUID(ctx context.Context, uid uint) (*userapi.UserInfoData, error) {
	ret := userapi.UserInfoData{}
	u, err := getRepo().GetByUID(ctx, uid)
	if err != nil {
		return nil, errs.ErrDB(err)
	}
	if u == nil {
		return nil, nil
	}
	ret.UID = u.ID
	ret.Name = u.UserName
	ret.AccountID = u.AccountID
	ret.Email = u.Email

	return &ret, nil
}

// doDummy 演示用户领域反向调用 Self 领域提供的 GetPing 能力。
func doDummy(ctx context.Context, id int) (userapi.DummyData, error) {
	pingGetter := getDomain().pingGetter
	if pingGetter == nil {
		return userapi.DummyData{}, errors.New("ping getter尚未注入")
	}
	if _, err := pingGetter.GetPing(ctx); err != nil {
		return userapi.DummyData{}, err
	}
	return userapi.DummyData{Echo: id + 1}, nil
}
