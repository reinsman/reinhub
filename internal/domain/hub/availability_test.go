package hub

import (
	"context"
	"sync"
	"testing"

	pb "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAgentDownIsOwnedIdempotentAndPreservesHistory(t *testing.T) {
	d, path := setupHub(t)
	ctx := context.Background()
	request := &pb.UnregisterAgentRequest{RequestId: "down-1", DaemonId: "daemon", AgentId: "agent"}
	wrong := &pb.UnregisterAgentRequest{RequestId: "wrong", DaemonId: "other", AgentId: "agent"}
	if _, err := d.UnregisterAgent(ctx, wrong); status.Code(err) != codes.PermissionDenied {
		t.Fatal("wrong owner accepted", err)
	}
	created, err := d.CreateRun(ctx, &pb.CreateRunRequest{RequestId: "turn", AgentId: "agent", Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = d.UnregisterAgent(ctx, request); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("queued run did not prevent down", err)
	}
	claimed, err := d.ClaimRuns(ctx, &pb.ClaimRunsRequest{RequestId: "claim", DaemonId: "daemon", RuntimeIds: []string{"runtime"}, Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	run := claimed.Leases[0].Run
	if _, err = d.UnregisterAgent(ctx, request); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("dispatched run did not prevent down", err)
	}
	if _, err = d.record(ctx, "daemon", &pb.ExecutionEvent{Attempt: &pb.AttemptRef{RunId: run.Id, AttemptId: run.AttemptId}, Sequence: 1, Payload: &pb.ExecutionEvent_Result{Result: &pb.ResultEvent{State: pb.RunState_RUN_STATE_COMPLETED, ExecutionStopped: true}}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = d.UnregisterAgent(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
	agents, err := d.ListAgents(ctx, &pb.ListAgentsRequest{})
	if err != nil || len(agents.Agents) != 1 || !agents.Agents[0].Agent.Disabled || agents.Agents[0].Online {
		t.Fatal("offline Agent must remain visible", agents, err)
	}
	if _, err = d.CreateRun(ctx, &pb.CreateRunRequest{RequestId: "disabled", AgentId: "agent", Prompt: "must not execute"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("offline Agent accepted a run", err)
	}
	if _, err = d.RegisterAgent(ctx, &pb.RegisterAgentRequest{RequestId: "up-2", DaemonId: "daemon", Agent: &pb.Agent{Id: "agent", RuntimeId: "runtime", Name: "Codex"}}); err != nil {
		t.Fatal(err)
	}
	// 重试旧下线事务不能覆盖后续明确的上线操作。
	if _, err = d.UnregisterAgent(ctx, request); err != nil {
		t.Fatal(err)
	}
	agents, _ = d.ListAgents(ctx, &pb.ListAgentsRequest{})
	if agents.Agents[0].Agent.Disabled {
		t.Fatal("old down request disabled a later up")
	}
	if _, err = d.UnregisterAgent(ctx, &pb.UnregisterAgentRequest{RequestId: request.RequestId, DaemonId: "daemon", AgentId: "different"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("request ID conflict accepted", err)
	}
	if _, err = d.UnregisterAgent(ctx, &pb.UnregisterAgentRequest{RequestId: "down-2", DaemonId: "daemon", AgentId: "agent"}); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewHub(path)
	if err != nil {
		t.Fatal(err)
	}
	agents, err = reopened.ListAgents(ctx, &pb.ListAgentsRequest{})
	if err != nil || !agents.Agents[0].Agent.Disabled {
		t.Fatal("down state not durable", err)
	}
	history, err := reopened.history(ctx, "agent", created.Run.SessionId, "", 20)
	if err != nil || len(history.Runs) != 1 || len(history.Runs[0].Events) != 1 {
		t.Fatal("history lost after down", err)
	}
}

func TestConcurrentDownAndCreateKeepSchedulingAtomic(t *testing.T) {
	d, _ := setupHub(t)
	ctx := context.Background()
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, _ = d.CreateRun(ctx, &pb.CreateRunRequest{RequestId: "race-run", AgentId: "agent", Prompt: "hello"})
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _ = d.UnregisterAgent(ctx, &pb.UnregisterAgentRequest{RequestId: "race-down", DaemonId: "daemon", AgentId: "agent"})
	}()
	close(start)
	wg.Wait()
	if err := d.repo.view(ctx, func(s localState) error {
		if s.Agents["agent"].Disabled && len(s.Runs) != 0 {
			t.Fatal("a run was queued on a disabled Agent")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
