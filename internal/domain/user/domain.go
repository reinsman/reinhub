package user

import (
	"context"
	"errors"

	domaincontract "github.com/reinsman/reinhub/internal/domain"

	"gorm.io/gorm"
)

var currentDomain *Domain

// Domain 保存用户领域在当前进程中的领域级依赖。
// 每个进程只允许初始化一个用户领域实例。
type Domain struct {
	repo       Repository
	pingGetter domaincontract.PingGetter
}

// InitDomain 初始化当前进程唯一的用户领域。
func InitDomain(db *gorm.DB) *Domain {
	return initDomain(NewRepository(db))
}

// initDomain 使用指定 Repository 初始化用户领域，供包内测试替换数据实现。
func initDomain(repo Repository) *Domain {
	if currentDomain != nil {
		panic("用户领域重复初始化")
	}
	currentDomain = &Domain{
		repo: repo,
	}
	return currentDomain
}

// getDomain 获取当前进程的用户领域，仅供包内代码使用。
func getDomain() *Domain {
	if currentDomain == nil {
		panic("用户领域尚未初始化")
	}
	return currentDomain
}

// getRepo 获取用户领域的 Repository，仅供包内 Service 函数使用。
func getRepo() Repository {
	return getDomain().repo
}

// WithPingGetter 在应用启动阶段注入 Self 领域提供的 Ping 能力。
func (d *Domain) WithPingGetter(getter domaincontract.PingGetter) error {
	if getter == nil {
		return errors.New("ping getter不能为空")
	}
	if d.pingGetter != nil {
		return errors.New("ping getter已经注入")
	}
	d.pingGetter = getter
	return nil
}

// GetUserNum 对其他领域公开用户数量查询能力。
func (d *Domain) GetUserNum(ctx context.Context) (int, error) {
	return getUserNum(ctx)
}
