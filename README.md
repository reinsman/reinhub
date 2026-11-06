# reinhub

基于 Gin、GORM 和 MySQL 的 Go 服务脚手架，采用“领域优先 + 分层”的组织方式。

项目提供显式依赖组装、领域路由注册、数据库模型迁移、统一响应、错误码、JWT 鉴权、访问日志、Swagger、优雅退出及部署示例。

## 初始化新项目

从脚手架创建项目后，先使用目标 Go module 初始化项目：

```bash
./initmodule.sh github.com/your-org/your-service
```

该命令会更新 `go.mod` 和项目内的 Go import 路径。服务名取 module 的最后一段，`build.sh` 和 `run.sh` 会据此生成并启动对应名称的二进制。镜像仓库、部署资源名和 Swagger 服务信息属于部署配置，请按实际项目单独修改。

## 编译

环境要求：

- Go 1.25（推荐使用 `go.mod` 指定的 Go 1.25.1 toolchain）
- 运行服务时需要 MySQL 8.x

执行编译脚本：

```bash
./build.sh
```

编译产物位于 `bin/reinhub`。

## 部署

构建镜像：

```bash
./deploy/image/build-image.sh
```

构建并推送镜像，可选语义版本标签：

```bash
./deploy/image/build-image.sh -p -v 1.0.0
```

镜像名称和仓库地址配置在 `deploy/image/build-image.sh`。Kubernetes 部署前，请同步更新 `deploy/kubernetes/deployment.yaml` 中的镜像标签及环境配置。

```bash
./deploy/kubernetes/install.sh
./deploy/kubernetes/uninstall.sh
```

也可以使用 Helm 安装；安装前请覆盖示例 JWT、数据库密码和数据库地址，具体参数及 Secret、持久化配置参阅 [deploy/helm/README.md](deploy/helm/README.md)：

```bash
helm upgrade --install examplesrv ./deploy/helm \
  --namespace iam \
  --create-namespace
```

## 开发使用说明

本地运行、目录结构、架构约束、领域开发、跨领域协作、Swagger 和测试规范请参阅 [DEVELOPMENT.md](DEVELOPMENT.md)。

## Hub gRPC 框架

当前阶段沿用模板的 Gin/GORM/MySQL 和应用生命周期，在 `internal/domain/hub` 增加 Hub 领域，`internal/app/grpc_server.go` 组装四个生成服务。已移除模板的链路追踪、指标 SDK、中间件、HTTP instrumentation 及对应配置；保留 slog、请求 ID 和 HTTP 示例。

开发时需要并排检出 `../proto`，Go 使用本地 replace。协议、生成检查和状态语义均在 proto 仓库，禁止在 Hub 手写另一套消息类型。发布时固定协议版本并移除本地 replace。

在 `bin/conf.toml` 配置：

```toml
[grpc]
listen = "127.0.0.1:9011"
cert_file = "/absolute/path/to/server.crt"
key_file = "/absolute/path/to/server.key"
```

listen 为空时保留原 HTTP 模板运行方式；启用 gRPC 必须提供有效 TLS 证书，不支持明文降级。程序仍通过模板配置连接 MySQL。Hub 新模型尚未接入迁移，不会因本轮骨架自动创建 Hub 业务表。

`StartLogin/GetLoginStatus` 返回 UNIMPLEMENTED；其他 RPC 与双向流在真实认证器完成前默认拒绝。模板原有 `_HCID` HTTP 示例不是 Hub 认证机制，部署前应移除示例路由或限定在可信网关之后。

- [总体架构与当前实现边界](../proto/docs/architecture.md)
- [状态、权限、幂等、ACK、取消与恢复](../proto/docs/semantics.md)

```sh
go test ./...
go test -race ./...
```

当前仅验证服务注册、错误边界和框架编译；登录、队列、租约、数据库事务、控制流和端到端执行仍为 TODO。
