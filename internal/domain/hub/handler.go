package hub

import (
	"context"

	hubv1 "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *Domain) StartLogin(context.Context, *hubv1.StartLoginRequest) (*hubv1.StartLoginResponse, error) {
	// TODO: 校验 daemon/login_secret，存摘要与有效期，创建浏览器授权流程。
	return nil, status.Error(codes.Unimplemented, "Hub login flow is not implemented")
}

func (d *Domain) GetLoginStatus(context.Context, *hubv1.GetLoginStatusRequest) (*hubv1.GetLoginStatusResponse, error) {
	// TODO: 校验请求所有权；仅授权完成后签发绑定用户和 daemon 的凭据。
	return nil, status.Error(codes.Unimplemented, "Hub credential issuance is not implemented")
}

func (d *Domain) RegisterRuntime(context.Context, *hubv1.RegisterRuntimeRequest) (*hubv1.RegisterRuntimeResponse, error) {
	// TODO: 校验 workspace 权限和 daemon 绑定，按 request_id 幂等注册能力。
	return nil, status.Error(codes.Unimplemented, "runtime registration is not implemented")
}

func (d *Domain) CreateRun(context.Context, *hubv1.CreateRunRequest) (*hubv1.CreateRunResponse, error) {
	// TODO: 校验 Agent/Runtime 权限，事务创建 Session/queued Run，提交后通知。
	return nil, status.Error(codes.Unimplemented, "run submission is not implemented")
}

func (d *Domain) ClaimRuns(context.Context, *hubv1.ClaimRunsRequest) (*hubv1.ClaimRunsResponse, error) {
	// TODO: Repository 事务领取；request_id 去重，按容量建立 attempt 与启动租约。
	return nil, status.Error(codes.Unimplemented, "run claiming is not implemented")
}

func (d *Domain) GetRun(context.Context, *hubv1.GetRunRequest) (*hubv1.GetRunResponse, error) {
	// TODO: 先授权，再读取平台权威状态；失联不能推断执行已终止。
	return nil, status.Error(codes.Unimplemented, "run lookup is not implemented")
}

func (d *Domain) RequestCancel(context.Context, *hubv1.RequestCancelRequest) (*hubv1.RequestCancelResponse, error) {
	// TODO: 持久化取消意图与 command_id，等待执行停止确认后才能标记 cancelled。
	return nil, status.Error(codes.Unimplemented, "cancellation requests are not implemented")
}

func (d *Domain) ReplyInteraction(context.Context, *hubv1.ReplyInteractionRequest) (*hubv1.ReplyInteractionResponse, error) {
	// TODO: 校验用户权限、attempt、interaction_id 和回复结构，事务写入控制 outbox。
	return nil, status.Error(codes.Unimplemented, "interaction replies are not implemented")
}

func (d *Domain) SendInstruction(context.Context, *hubv1.SendInstructionRequest) (*hubv1.SendInstructionResponse, error) {
	// TODO: 校验 Runtime 专属能力，持久化命令；不将 Native 能力降为 A2A 子集。
	return nil, status.Error(codes.Unimplemented, "additional instructions are not implemented")
}

func (d *Domain) Connect(grpc.BidiStreamingServer[hubv1.ConnectRequest, hubv1.ConnectResponse]) error {
	// TODO: Hello → 对账 → 双向收发；事件事务提交后累计 ACK，命令持久化投递。
	// TODO: 心跳、撤销检测、attempt fencing、按序事件入库和终态去重。
	return status.Error(codes.Unimplemented, "daemon event stream is not implemented")
}
