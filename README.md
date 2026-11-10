# reinhub

Rein 的 Go 服务端，向 Console 提供 Agent 与执行记录接口，通过 gRPC/TLS 与 reind 交互。Hub 保存平台权威状态；原生 Agent 子进程和 A2A 服务调用由 reind 管理。

```text
rein CLI → Unix socket → reind → gRPC/TLS → reinhub
Console → HTTP API → reinhub

reind
  ├─ Native Backend → stdin/stdout → Codex / Pi
  └─ A2A Backend → HTTP/A2A → Agent 服务
```

当前实现支持 Console → Hub → daemon → Agent 的执行闭环。

## 本机启动

环境要求：

- Go 1.25，toolchain 为 Go 1.25.1，见 `go.mod`。
- 并排检出协议仓库 `../proto`；当前通过 Go `replace` 使用其中的生成代码。
- 当前不依赖 MySQL，暂不接入登录。

在仓库根目录运行：

```sh
go run ./cmd/server -conf bin/conf.toml
```

也可以先编译再启动：

```sh
./build.sh
./bin/reinhub -conf bin/conf.toml
```

配置示例：

```toml
listen = "127.0.0.1:9010"
loglv = "info"
directory = "~/.reinhub"

[grpc]
listen = "127.0.0.1:9011"
```

HTTP 与 gRPC 监听地址必须为数字 loopback 地址。首次启动会在当前用户的 `~/.reinhub/` 生成本机 CA、服务端证书和私钥；reind 默认使用 `--ca ca.pem` 校验 Hub，不修改系统信任目录；也可显式使用 `--insecure-skip-verify` 保留 TLS 加密并跳过证书校验。`directory` 省略或为空时默认使用 `~/.reinhub`，支持展开 `~/`；显式相对路径仍相对于进程工作目录解析。

原 `[agent_development]` 配置已移除：请删除 `enabled`，将 `directory` 移至 TOML 顶层（放在 `[grpc]` 前）。旧配置会报错提示，不会静默切换数据目录。

已有 `.local-agents/` 数据不会自动迁移。若需继续使用原数据和证书，可显式保留 `directory = ".local-agents"`；切换到新的空目录会生成新证书，需要更新 reind 的 `--ca` 路径。

启动后可检查：

```sh
curl http://127.0.0.1:9010/api/health
```

随后通过 `rein register agent codex`、`rein register agent pi` 或 `rein register agent a2a` 注册 Agent，再打开 Console。注册参数、CA 配置、Console 代理和端到端验证见 [跨仓库启动说明](../rein/docs/console-integration.md)。

## 当前能力

- Agent 上下线由 CLI `rein up` / `rein down` 经 daemon 同步。下线保留描述、稳定 ID 和历史，列表显示 disabled=true、online=false，禁止创建和领取新任务；有未结束 Run 时拒绝下线。
- 接收 reind 的 Runtime、Agent 注册，向 Console 返回 Agent 列表与 daemon 在线状态。
- 创建 Run、排队、按容量领取，使用 attempt 和租约校验执行归属。
- 通过双向 gRPC 流处理心跳、工作通知、执行事件、交互回复、取消命令及确认。
- 保存执行记录、事件与控制回执；事件去重、持久化后 ACK，断线后重连对账。
- 提供 Console HTTP API：Agent 列表、最近对话与历史分页、Run 创建与查询、审批回复和取消；通过 SSE 推送 Run 实时增量，断线按持久化游标重放。

Hub 使用单进程文件锁和原子快照，将权威状态保存在 `~/.reinhub/hub-state.json`。当前上限为 512 个 Run、64 MiB 快照，适用于本机验证。

## 协议与代码位置

Protobuf 与 gRPC 协议唯一来源是独立的 [proto 仓库](../proto)。禁止手改生成代码或在 Hub 维护另一套协议类型；发布前需将本地 `replace` 改为固定协议版本。

- `cmd/server/`：配置、日志、信号与进程生命周期。
- `internal/app/`：HTTP/gRPC 服务、TLS 与依赖组装。
- `internal/domain/hub/`：注册、队列、控制流、状态与存储。
- `pkg/api/v1/hub/`：Console HTTP 请求与响应类型。
- [架构与实现边界](../proto/docs/architecture.md)、[状态与交互语义](../proto/docs/semantics.md)。

## 验证

健康检查、注册列表、创建/查询 Run、幂等、审批、取消与错误边界的可复制命令见 [curl 测试文档](TESTING.md)。

```sh
go test ./...
go test -race ./...
```

跨仓库测试还覆盖原生/A2A 执行、会话续接、审批、取消及重启恢复，运行方式见 [跨仓库启动说明](../rein/docs/console-integration.md#自动化验证)。

## 实现边界

当前阶段只有这一套 Hub 实现：文件存储、自动生成 TLS 证书、暂不接入登录，并仅支持本机 loopback 监听。没有开发/生产模式切换，也不通过 `-debug` 选择实现。

认证授权、凭据签发与续期、事务数据库和多实例运行尚未实现；待 daemon 功能完备后再重构 Hub。

底层开发约定见 [DEVELOPMENT.md](DEVELOPMENT.md)。保留的模板领域和数据库工具仅供后续重构参考，不作为另一个服务启动入口。当前使用 slog 与请求 ID，不接入链路追踪或指标 SDK。
