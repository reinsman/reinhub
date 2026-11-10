# reinhub 开发指南

当前 Hub 使用文件存储和 gRPC/TLS，暂不接入登录。下文保留部分 Gin/GORM 模板开发约定与业务示例，供后续重构参考；它们不是另一种 Hub 运行模式。当前启动、配置和实现边界以 [README.md](README.md) 为准。

## 开发环境

- Go 1.25（推荐使用 `go.mod` 指定的 Go 1.25.1 toolchain）
- 当前 Hub 不依赖 MySQL；数据库工具与模板示例仍保留。
- 可选：Docker、Docker Compose、swag CLI

## 本地运行

并排检出 `../proto`，在仓库根目录运行：

```bash
go run ./cmd/server -conf bin/conf.toml

# 或编译后使用同一配置启动
./build.sh
./bin/reinhub -conf bin/conf.toml
```

默认 HTTP 为 `127.0.0.1:9010`，gRPC/TLS 为 `127.0.0.1:9011`，健康检查为 `GET /api/health`。顶层 `directory` 默认使用 `~/.reinhub`，保存证书和权威状态；无须开启额外模式开关。

## 项目结构

```text
.
├── cmd/server/                 程序入口：配置、日志、信号和应用启动
├── internal/
│   ├── app/
│   │   ├── app.go             应用初始化、运行和进程级资源关闭
│   │   ├── routers.go         领域依赖组装与路由注册
│   │   ├── http_server.go     HTTP Server 创建与优雅退出
│   │   └── middleware/        鉴权、访问日志和 Recovery
│   ├── config/                TOML 配置加载
│   ├── dbs/                   多种数据库连接、模型注册与持久化辅助
│   ├── domain/                领域能力契约与实现
│   │   ├── self.go            Self 领域对内开放的能力接口
│   │   ├── user.go            User 领域对内开放的能力接口
│   │   ├── self/              健康检查与 Swagger 等服务自省实现
│   │   └── user/              用户领域与参数绑定示例实现
│   ├── ginutil/               Gin Context 与统一响应辅助
│   ├── globals/               构建版本等进程级信息
│   └── testutil/              测试辅助
├── pkg/
│   ├── api/                   可供其他微服务引用的公开契约
│   │   ├── errs/              通用错误码与 CodeErr
│   │   ├── rsp/               统一响应体 Rsp/BaseRsp
│   │   └── v1/                按版本和领域组织的 API 类型
│   │       ├── ping/
│   │       └── user/
│   ├── crypto/                加密与密码哈希
│   ├── graceful/              优雅退出工具
│   ├── hashid/                HashID 编解码
│   ├── httpx/                 HTTP 客户端工具
│   ├── jwt/                   JWT 生成与解析
│   ├── nulltypes/             SQL Null 类型转换
│   ├── osutil/                操作系统辅助
│   ├── retry/                 重试工具
│   └── strutil/               字符串与解析工具
├── docs/                      Swagger 生成物
├── deploy/                    Compose、镜像和 Kubernetes 部署文件
├── bin/                       编译产物与本地运行配置
├── build.sh                   编译脚本
├── run.sh                     本地运行脚本
└── swag.sh                    Swagger 生成脚本
```

## 架构与依赖方向

> **重要：按业务复杂度选择架构。** 本项目主要用于微服务，单个服务的业务边界通常已经较窄。如果业务简单、职责单一，应优先在一个 Domain 内完成开发，无需为了形式上的领域分层而拆分多个 Domain、增加跨领域接口或引入额外组装逻辑。只有当业务中出现明确、稳定且需要独立演进的领域边界时，才应拆分领域并承担相应的分层成本。以下多领域依赖和协作约定仅在实际需要时适用。

```text
cmd/server
    ├── internal/config
    └── internal/app
            ├── internal/config
            ├── internal/dbs
            ├── internal/app/middleware
            ├── internal/ginutil ──→ pkg/api
            ├── internal/domain ──→ pkg/api
            └── internal/domain/*
                    ├── internal/domain
                    ├── internal/dbs
                    ├── internal/ginutil
                    └── pkg/api
```

基本约束：

- `app.New` 负责组装 Hub 文件存储、HTTP API 和 gRPC/TLS 服务。
- `Application.Run` 负责 HTTP/gRPC 服务运行与优雅退出，返回时释放存储锁。
- `cmd/server` 只负责加载配置、初始化日志、信号与启动应用。
- `internal/config` 只负责配置结构、解析与默认值，不保存全局配置状态。
- 领域模型在包初始化阶段通过 `dbs.RegisterModels` 注册，建立数据库连接后由 `dbs.AutoMigrate` 统一迁移。
- `internal/dbs` 集中管理关系型及非关系型数据库能力；当前只保存待迁移模型的接口值，不导入也不识别具体领域。
- 每个领域通过 `InitDomain` 初始化当前进程唯一的领域实例，领域级 Repository 和跨领域依赖保存在 `Domain` 中。
- `internal/domain` 按领域集中定义对内开放的窄接口，可以引用 `pkg/api` 契约，但不能反向导入任何领域实现子包。
- `domain/*/handler.go` 使用不导出的包级函数处理 HTTP 参数、上下文、Service 调用和响应。
- `service.go` 使用不导出的包级函数承载业务编排，通过 `getDomain`、`getRepo` 读取领域级依赖；Service 函数接收标准库 `context.Context`，不依赖 `gin.Context`。
- `repository.go` 只处理数据访问，由 `InitDomain` 创建并保存到 Domain，数据访问时使用 `db.WithContext(ctx)` 传递请求上下文。
- `model.go` 映射数据库表，不直接作为 HTTP 响应。
- `internal/app/middleware` 实现并统一挂载全局 Gin 中间件。
- `internal/ginutil` 提供 Handler 和 Middleware 共享的 Gin Context 与统一响应能力，不依赖 `app` 或具体领域。
- 每个领域的 `routes.go` 只声明路径、HTTP 方法和 Handler 映射；`internal/app/router.go` 负责初始化领域并调用路由注册函数。
- `pkg/api/v1/<domain>` 保存稳定、版本化的对外请求与响应类型。
- `pkg/api` 不依赖 `internal`；其他微服务可以安全引用其中的契约。
- `internal/ginutil`、`internal/app/middleware` 和 `internal/dbs` 不反向依赖具体领域。
- 领域包不依赖 `internal/app`，避免循环依赖。

领域 Handler 使用点导入引用 Gin 辅助包，因此统一响应可以直接写为：

```go
import . "github.com/iam/examplesrv/internal/ginutil"

Done(c, data, err)
```

### 上下文传递

Handler 从 HTTP 请求中取出标准库 Context，并沿包级 Service 函数、Repository 向下传递：

```go
func userInfo(c *gin.Context) {
	ctx := c.Request.Context()
	data, err := getByUID(ctx, uid)
	Done(c, data, err)
}

func getByUID(ctx context.Context, uid uint) (*UserInfo, error) {
	return getRepo().GetByUID(ctx, uid)
}

func (r *gormRepository) GetByUID(ctx context.Context, uid uint) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).First(&user, uid).Error
	return &user, err
}
```

统一将 `context.Context` 作为 Service 函数的第一个参数。不在 Service 或 Repository 中创建 `context.Background()` 替换请求 Context，也不将 `*gin.Context` 传入 Service。这样请求取消、超时和链路信息可以继续传递到数据库操作。

### 请求 ID 传递

访问日志中间件会从请求头读取请求 ID。请求头名称由环境变量 `REQUEST-ID-HEADER` 指定，未配置或值为空时默认使用 `x-request-id`。

中间件会在调用 Handler 前将请求 ID 写入 `c.Request.Context()`，并在请求处理完成后以 `request_id` 字段写入访问日志。Service 函数将 Context 传给 `pkg/httpx` 发起下游请求时，`httpx` 会自动使用同名请求头传播该 ID：

```go
err := httpx.HttpPost(endpoint, path, body, ctx)
```

如果入站请求未携带该请求头，请求 ID 为空；当前脚手架不自动生成新的 ID。

### 日志

保留 slog JSON 日志、请求 ID 与访问日志。Hub 的 gRPC 请求使用标准 status；不记录登录秘密、连接凭据或原生模型凭据。

## 跨领域协作

Handler 只处理输入输出并调用所属领域的 Service 函数，不直接编排其他领域。跨领域能力接口集中定义在 `internal/domain`，具体实现由 `internal/app` 在启动时注入。

每个领域对应一个契约文件，并按稳定用途拆分小接口。例如多个领域需要判断用户是否有效：

```go
// internal/domain/user.go
type UserChecker interface {
	UserActive(ctx context.Context, uid uint) (bool, error)
}
```

`user.Domain` 只要具有同签名方法就会自动实现该接口。Order 等调用方只依赖 `domain.UserChecker`，不需要了解 User 领域的 Repository 或内部 Service 函数。一个文件可以按用途定义 `UserReader`、`UserCounter`、`UserChecker` 等多个小接口，不要把一个领域的所有方法合并成巨型接口。

领域实现统一依赖契约包：

```text
internal/domain/user  ──→ internal/domain
internal/domain/self  ──→ internal/domain
```

`internal/domain` 不能反向导入 `internal/domain/user`、`internal/domain/self` 等实现包。接口参数和返回值可以使用基础类型、标准库类型或 `pkg/api` 中的稳定契约类型。

具体领域将需要的接口保存在自己的 Domain 中：

```go
// internal/domain/order/domain.go
type Domain struct {
	users domain.UserChecker
}
```

接口集中定义既避免调用方重复声明，也让能力提供方统一维护语义；调用方仍应只依赖符合用途的最小接口。

### 运行时双向依赖

Go 的包导入图必须无环，但对象引用图可以有环。本项目的 User 和 Self 领域通过 `internal/domain` 集中定义的窄接口形成运行时双向依赖：

```go
userDomain := user.InitDomain(db)
selfDomain := self.InitDomain(userDomain)

if err := userDomain.WithPingGetter(selfDomain); err != nil {
	panic(err)
}
```

其中 `self.GetPing` 会调用 `user.GetUserNum`，`user.DoDummy` 会反向调用 `self.GetPing`。`GetUserNum` 是不再反调 Self 的叶子能力，因此调用链不会无限递归。两个领域包都不导入对方，具体对象只在 `internal/app` 中关联。

这种方式解决了对象的构造顺序，最终的运行时引用仍然是双向的。运行时双向引用并非天然错误，但应满足以下条件：

- 双方依赖的是 `internal/domain` 中集中定义的窄接口，而不是对方的具体 Domain 类型。
- `WithXXX` 只在 `internal/app` 的启动组装阶段调用，不在业务运行期替换依赖。
- Setter 拒绝 `nil` 和重复设置，应用启动前保证 Domain 已完整组装。
- 双方的方法调用不形成递归调用链，也不在持有内部锁时调用对方。
- 资源的启动和关闭仍由 `Application` 统一管理。

对象相互持有不等于一定会出现递归或死锁；真正需要审查的是方法调用链、锁顺序和初始化状态。

当双向依赖持续扩张、责任边界变得模糊时，优先将基础能力与跨领域组合能力拆分：

```text
user.CoreService
        ↓
  order.Service
        ↓
user.ProfileService
```

拆分后可按单向顺序组装：

```go
userCore := user.NewCoreService(db)
orderService := order.NewService(db, userCore)
userProfile := user.NewProfileService(userCore, orderService)
```

如果一个用例需要同时修改多个领域，可以提取独立的业务流程 Service 进行编排，避免两个领域 Service 之间的协作持续扩张。内部 Service 接口不放入 `pkg/api`；`pkg/api` 只保存对外 API 契约。

基本约束：

- 跨领域能力接口集中放在 `internal/domain`，按领域拆分文件，并按稳定用途定义小接口。
- `internal/domain` 只依赖标准库或 `pkg/api`，不能依赖具体领域实现。
- 领域 Domain 可以通过窄接口形成受控的运行时双向依赖，但包导入图仍必须无环。
- 延后注入必须由 `internal/app` 集中完成，不使用全局变量构造循环。
- 当双向协作导致测试组装、调用链或事务边界难以理解时，拆分 Service 或提取独立流程。

## 新增领域接口

以 `order` 领域为例：

1. 在 `pkg/api/v1/order/types.go` 定义稳定的请求和响应契约。
2. 在 `internal/domain/order/model.go` 定义数据库模型，并在 `init` 中调用 `dbs.RegisterModels`。
3. 在 `repository.go` 定义 Repository 接口及 GORM 实现，并由 `InitDomain` 保存领域唯一实例。
4. 在 `service.go` 使用不导出的包级函数实现业务逻辑，第一个参数使用 `context.Context`，通过 `getRepo` 获取数据访问能力。
5. 在 `handler.go` 使用不导出的包级函数完成参数绑定，将 `c.Request.Context()` 传入 Service 函数，并返回统一响应。
6. 在领域的 `routes.go` 声明路由，并在 `internal/app/router.go` 组装依赖后调用。
7. 将新 Handler 和契约目录加入 `swag.sh`，重新生成文档。
8. 补充 Service 函数单元测试和必要的接口测试；测试结束后重置包级 Domain，避免用例互相污染。
9. Hub 协议修改应先更新独立 proto 仓库并重新生成 Rust/Go 接口。

## API 契约

### Service 响应数据约定

直接为 Handler 生成 HTTP 响应 `data` 的 Service 函数，返回值必须使用 `pkg/api/v1/<domain>` 中定义的响应 DTO。DTO 必须是结构体，Service 统一返回该结构体的指针，不返回结构体值、`any`、数据库 Model 或领域内部对象：

```go
func getByUID(ctx context.Context, uid uint) (*userapi.UserInfoData, error) {
	user, err := getRepo().GetByUID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &userapi.UserInfoData{
		UID:  user.ID,
		Name: user.UserName,
	}, nil
}
```

返回值遵循以下约定：

- 成功且有数据时返回 `&Data{...}, nil`。
- 失败时返回 `nil, err`，使失败响应中的 `data` 为 `null`；不返回 `&Data{}, err`。
- 除非 API 语义明确允许“成功但无数据”，否则不返回 `nil, nil`。
- 列表结果也应封装为 `pkg/api` 结构体并返回指针，不直接返回 `*[]T`。
- 没有响应数据的操作可以只返回 `error`；如果协议要求必须存在 `data`，则在 `pkg/api` 中定义空响应 DTO。

例如列表响应定义为：

```go
type UserListData struct {
	Users []UserInfoData `json:"users"`
	Total int64          `json:"total"`
}

func listUsers(ctx context.Context) (*userapi.UserListData, error)
```

Handler 不再转换 Service 返回的 DTO，直接交给统一响应层：

```go
data, err := getByUID(c.Request.Context(), uid)
Done(c, data, err)
```

`Rsp.Data` 当前的类型为 `any`。当具体指针为 `nil` 并被放入 `any` 后，该接口值本身不等于 `nil`，但 JSON 序列化结果仍为 `null`。响应处理代码不应使用 `rsp.Data == nil` 判断具体指针是否为空。

该约定只适用于直接产生 HTTP `data` 的 Service 函数。不直接对应 HTTP 响应的领域内部计算和跨领域窄接口，可以按语义返回 `bool`、计数值等基础类型。

`pkg/api/v1/<domain>` 之间的依赖也必须保持无环；出现循环引用时，按前文 Service 跨领域循环依赖的同样思路处理：将共享契约下沉到独立的基础包，或将组合响应放到更高层的用例包，不允许同级领域 API 包相互引用。

### HTTP 状态码与业务错误码

以下 HTTP 200 约定适用于保留的 Gin 模板示例。当前 Hub Console API 使用标准 HTTP 状态码，错误响应为 `{"error":"..."}`，协议消息使用 protobuf JSON；这是已有 Console 契约，curl 用例见 [TESTING.md](TESTING.md)。Hub gRPC 使用标准 gRPC status。

已识别并正常封装的业务响应统一返回 HTTP 200，包括参数错误、鉴权失败、数据库错误和领域业务错误。客户端必须通过响应体的 `code` 判断调用结果：

- `code == 0`：成功。
- `code != 0`：失败，具体含义由错误码定义决定。

HTTP 404、未捕获 panic 导致的 HTTP 500 等未进入正常业务响应流程的协议或系统级错误，仍可以使用非 200 HTTP 状态码。

公共响应格式位于 `pkg/api/rsp`：

```json
{
  "code": 0,
  "msg": "成功",
  "data": {},
  "_cost": 1,
  "_err": ""
}
```

其中 `_cost` 和 `_err` 仅用于调试；Release 模式会隐藏这些字段的内容。通用错误码位于 `pkg/api/errs`，领域专属错误码应使用独立区间，避免跨服务冲突。

## Swagger

安装 `swag` 后执行：

```bash
./swag.sh
```

生成文件位于 `docs/docs.go`、`docs/swagger.json` 和 `docs/swagger.yaml`。
`swag.sh` 会自动发现 `internal/domain` 和 `pkg/api` 下包含 Go 源码的包，新增领域时不需要再修改脚本中的目录列表。

## 测试

领域测试优先使用 fake Repository，不连接 MySQL 或其他外部服务：

- `service_test.go` 验证业务分支、返回数据和错误转换。
- `handler_test.go` 使用 Gin TestMode 和 `httptest` 验证参数绑定与 API 响应。
- Repository 集成测试应使用 `*_integration_test.go` 命名，并通过 build tag 或环境变量显式启用。

```bash
# 只运行领域测试
go test ./internal/domain/...

# 运行全部单元测试；默认不依赖外部服务
go test ./...

# 使用竞态检测运行全部单元测试
go test -race ./...
```

`pkg/httpx` 使用 `httptest` 在测试进程内启动临时 HTTP Server，不需要预先启动外部测试服务。

[返回 README](README.md)
