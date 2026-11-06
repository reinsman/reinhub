// Package hub 是 daemon 控制面的领域入口，复用模板的 GORM 数据访问边界。
package hub

import (
	hubv1 "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// Domain 只实现协议骨架；未接入身份与事务前不返回业务成功。
type Domain struct {
	hubv1.UnimplementedAuthServiceServer
	hubv1.UnimplementedRuntimeServiceServer
	hubv1.UnimplementedRunServiceServer
	hubv1.UnimplementedDaemonServiceServer
	repository *Repository
}

func New(db *gorm.DB) *Domain {
	return &Domain{repository: &Repository{db: db}}
}

func (d *Domain) Register(server grpc.ServiceRegistrar) {
	hubv1.RegisterAuthServiceServer(server, d)
	hubv1.RegisterRuntimeServiceServer(server, d)
	hubv1.RegisterRunServiceServer(server, d)
	hubv1.RegisterDaemonServiceServer(server, d)
}
