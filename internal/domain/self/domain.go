package self

import (
	"context"

	domaincontract "github.com/reinsman/reinhub/internal/domain"
)

var currentDomain *Domain

// Domain 保存服务自省领域在当前进程中的领域级依赖。
// 每个进程只允许初始化一个 Self 领域实例。
type Domain struct {
	userCounter domaincontract.UserCounter
}

// InitDomain 初始化当前进程唯一的 Self 领域。
func InitDomain(userCounter domaincontract.UserCounter) *Domain {
	if currentDomain != nil {
		panic("Self领域重复初始化")
	}
	currentDomain = &Domain{
		userCounter: userCounter,
	}
	return currentDomain
}

// getDomain 获取当前进程的 Self 领域，仅供包内代码使用。
func getDomain() *Domain {
	if currentDomain == nil {
		panic("Self领域尚未初始化")
	}
	return currentDomain
}

// GetPing 对其他领域公开服务健康检查能力。
func (d *Domain) GetPing(ctx context.Context) (int, error) {
	return getPing(ctx)
}
