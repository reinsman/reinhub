package hub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func setupHub(t *testing.T) (*Hub, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	d, err := NewHub(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, err = d.RegisterRuntime(ctx, &pb.RegisterRuntimeRequest{Runtime: &pb.Runtime{Id: "runtime", DaemonId: "daemon", WorkspaceId: "workspace", Backend: pb.BackendKind_BACKEND_KIND_NATIVE, Provider: "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.RegisterAgent(ctx, &pb.RegisterAgentRequest{RequestId: "agent-reg", DaemonId: "daemon", Agent: &pb.Agent{Id: "agent", Name: "Codex", RuntimeId: "runtime"}})
	if err != nil {
		t.Fatal(err)
	}
	d.connections["daemon"] = &daemonConnection{wake: make(chan struct{}, 1), stop: make(chan struct{}), seen: time.Now()}
	return d, path
}
func createAndClaim(t *testing.T, d *Hub) *pb.Run {
	t.Helper()
	ctx := context.Background()
	created, err := d.CreateRun(ctx, &pb.CreateRunRequest{RequestId: "create-1", AgentId: "agent", Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	leases, err := d.ClaimRuns(ctx, &pb.ClaimRunsRequest{RequestId: "claim-1", DaemonId: "daemon", RuntimeIds: []string{"runtime"}, Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(leases.Leases) != 1 || leases.Leases[0].Run.Id != created.Run.Id {
		t.Fatal("run not claimed")
	}
	return leases.Leases[0].Run
}
func TestDurableEventsFencingAndIdempotency(t *testing.T) {
	d, path := setupHub(t)
	run := createAndClaim(t, d)
	ctx := context.Background()
	a := &pb.AttemptRef{RunId: run.Id, AttemptId: run.AttemptId}
	event := &pb.ExecutionEvent{Attempt: a, Sequence: 1, Payload: &pb.ExecutionEvent_State{State: &pb.StateEvent{State: pb.RunState_RUN_STATE_RUNNING}}}
	if _, err := d.record(context.Background(), "other", event); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("wrong daemon accepted: %v", err)
	}
	if _, err := d.record(context.Background(), "daemon", event); err != nil {
		t.Fatal(err)
	}
	if _, err := d.record(context.Background(), "daemon", event); err != nil {
		t.Fatal(err)
	}
	changed := proto.Clone(event).(*pb.ExecutionEvent)
	changed.Payload = &pb.ExecutionEvent_Text{Text: &pb.TextEvent{Text: "different"}}
	if _, err := d.record(context.Background(), "daemon", changed); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("conflicting event accepted")
	}
	gap := proto.Clone(event).(*pb.ExecutionEvent)
	gap.Sequence = 3
	if _, err := d.record(context.Background(), "daemon", gap); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("event gap accepted")
	}
	interaction := &pb.ExecutionEvent{Attempt: a, Sequence: 2, Payload: &pb.ExecutionEvent_Interaction{Interaction: &pb.InteractionEvent{InteractionId: "approval", ApprovalRequired: true, Method: "item/commandExecution/requestApproval", ParametersJson: []byte(`{}`)}}}
	if _, err := d.record(context.Background(), "daemon", interaction); err != nil {
		t.Fatal(err)
	}
	wait := &pb.ExecutionEvent{Attempt: a, Sequence: 3, Payload: &pb.ExecutionEvent_State{State: &pb.StateEvent{State: pb.RunState_RUN_STATE_WAITING_APPROVAL}}}
	if _, err := d.record(context.Background(), "daemon", wait); err != nil {
		t.Fatal(err)
	}
	reply := &pb.ReplyInteractionRequest{Attempt: a, CommandId: "reply", InteractionId: "approval", ResponseJson: []byte(`{"decision":"accept"}`)}
	if _, err := d.ReplyInteraction(ctx, reply); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ReplyInteraction(ctx, reply); err != nil {
		t.Fatal(err)
	}
	reply.CommandId = "duplicate-reply"
	if _, err := d.ReplyInteraction(ctx, reply); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("duplicate interaction accepted")
	}
	reopened, err := NewHub(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.GetRun(ctx, &pb.GetRunRequest{RunId: run.Id})
	if err != nil || got.Run.State != pb.RunState_RUN_STATE_WAITING_APPROVAL {
		t.Fatalf("state not durable: %v", err)
	}
	commands, err := reopened.pending(context.Background(), "daemon")
	if err != nil || len(commands) != 1 || commands[0].GetInteractionReply().InteractionId != "approval" {
		t.Fatal("oneof command not restored")
	}
	final := &pb.ExecutionEvent{Attempt: a, Sequence: 4, Payload: &pb.ExecutionEvent_Result{Result: &pb.ResultEvent{State: pb.RunState_RUN_STATE_CANCELLED}}}
	if _, err := reopened.record(context.Background(), "daemon", final); status.Code(err) != codes.InvalidArgument {
		t.Fatal("unconfirmed cancellation accepted")
	}
	final.GetResult().ExecutionStopped = true
	if _, err := reopened.record(context.Background(), "daemon", final); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.record(context.Background(), "daemon", final); err != nil {
		t.Fatal(err)
	}
	_ = reopened.repo.view(context.Background(), func(s localState) error {
		if len(s.Runs[run.Id].Events) != 4 {
			t.Fatal("duplicate event stored")
		}
		return nil
	})
}
func TestConcurrentClaimHasSingleOwnerAndSessionCannotCrossAgents(t *testing.T) {
	d, _ := setupHub(t)
	ctx := context.Background()
	_, err := d.CreateRun(ctx, &pb.CreateRunRequest{RequestId: "create", AgentId: "agent", Prompt: "hello", SessionId: "session"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	claimed := make(chan string, 2)
	for _, id := range []string{"one", "two"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := d.ClaimRuns(ctx, &pb.ClaimRunsRequest{RequestId: id, DaemonId: "daemon", RuntimeIds: []string{"runtime"}, Capacity: 1})
			if e != nil {
				t.Error(e)
				return
			}
			for _, lease := range r.Leases {
				claimed <- lease.Run.Id
			}
		}()
	}
	wg.Wait()
	close(claimed)
	if len(claimed) != 1 {
		t.Fatal("run claimed twice")
	}
	_, err = d.RegisterAgent(ctx, &pb.RegisterAgentRequest{RequestId: "reg2", DaemonId: "daemon", Agent: &pb.Agent{Id: "agent2", Name: "other", RuntimeId: "runtime"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.CreateRun(ctx, &pb.CreateRunRequest{RequestId: "cross", AgentId: "agent2", SessionId: "session", Prompt: "wrong"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatal("cross-agent session accepted")
	}
}
func TestConsoleAPIRejectsCrossSiteAndMalformedRequests(t *testing.T) {
	d, _ := setupHub(t)
	handler := d.ConsoleHandler()
	for _, test := range []struct {
		host, origin, content, body string
		code                        int
	}{{"evil.example", "", "application/json", "{}", 403}, {"localhost:9010", "https://evil.example", "application/json", "{}", 403}, {"localhost:9010", "", "text/plain", "{}", 415}, {"localhost:9010", "", "application/json", "{", 400}} {
		request := httptest.NewRequest("POST", "http://"+test.host+"/api/runs", strings.NewReader(test.body))
		request.Header.Set("Origin", test.origin)
		request.Header.Set("Content-Type", test.content)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != test.code {
			t.Fatalf("%+v got %d", test, recorder.Code)
		}
	}
	request := httptest.NewRequest("GET", "http://localhost:9010/api/agents", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	var payload struct {
		Agents []json.RawMessage `json:"agents"`
	}
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &payload) != nil || len(payload.Agents) != 1 {
		t.Fatal("Console registry unavailable")
	}
}
