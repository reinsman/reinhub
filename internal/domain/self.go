package domain

import "context"

// PingGetter 定义 Self 领域对其他领域提供的健康检查能力。
type PingGetter interface {
	GetPing(ctx context.Context) (int, error)
}
