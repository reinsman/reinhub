// Package domain 定义各领域对内提供的稳定能力契约。
package domain

import "context"

// UserCounter 定义用户领域对其他领域提供的用户统计能力。
type UserCounter interface {
	GetUserNum(ctx context.Context) (int, error)
}
