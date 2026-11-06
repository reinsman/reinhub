package self

import (
	"context"
	"errors"
)

// getPing 返回服务健康状态。
// 查询用户数量用于演示 Self 领域通过窄接口调用 User 领域。
func getPing(ctx context.Context) (int, error) {
	userCounter := getDomain().userCounter
	if userCounter == nil {
		return 0, errors.New("user counter尚未注入")
	}
	if _, err := userCounter.GetUserNum(ctx); err != nil {
		return 0, err
	}
	return 0, nil
}
