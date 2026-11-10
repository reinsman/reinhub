package hub

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type daemonConnection struct {
	wake chan struct{}
	stop chan struct{}
	seen time.Time
}

// Hub 提供当前阶段的注册、队列、执行记录和 daemon 控制流。
type Hub struct {
	*Domain
	repo        *localRepository
	mu          sync.Mutex
	connections map[string]*daemonConnection
}

func NewHub(path string) (*Hub, error) {
	r, err := openLocalRepository(path)
	if err != nil {
		return nil, err
	}
	return &Hub{Domain: New(nil), repo: r, connections: map[string]*daemonConnection{}}, nil
}
func (d *Hub) Register(s grpc.ServiceRegistrar) {
	pb.RegisterAuthServiceServer(s, d)
	pb.RegisterRuntimeServiceServer(s, d)
	pb.RegisterRunServiceServer(s, d)
	pb.RegisterDaemonServiceServer(s, d)
}
func invalid(s string) error  { return status.Error(codes.InvalidArgument, s) }
func conflict(s string) error { return status.Error(codes.FailedPrecondition, s) }
func terminal(s pb.RunState) bool {
	return s == pb.RunState_RUN_STATE_COMPLETED || s == pb.RunState_RUN_STATE_FAILED || s == pb.RunState_RUN_STATE_CANCELLED
}
func requestKey(v proto.Message) string {
	b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(v)
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
func (d *Hub) notify(daemon string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if c := d.connections[daemon]; c != nil {
		select {
		case c.wake <- struct{}{}:
		default:
		}
	}
}
func (d *Hub) online(daemon string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	c := d.connections[daemon]
	return c != nil && time.Since(c.seen) < 20*time.Second
}
func (d *Hub) RegisterRuntime(ctx context.Context, r *pb.RegisterRuntimeRequest) (*pb.RegisterRuntimeResponse, error) {
	v := r.GetRuntime()
	if v == nil || v.Id == "" || v.DaemonId == "" || v.WorkspaceId == "" || len(v.Id) > 128 || len(v.DaemonId) > 128 || len(v.WorkspaceId) > 128 || len(v.Capabilities) > 64 || (v.Backend != pb.BackendKind_BACKEND_KIND_NATIVE && v.Backend != pb.BackendKind_BACKEND_KIND_A2A) {
		return nil, invalid("runtime, daemon, workspace and backend are required")
	}
	err := d.repo.update(ctx, func(s *localState) error {
		if old := s.Runtimes[v.Id]; old != nil && old.DaemonId != v.DaemonId {
			return conflict("runtime owner cannot change")
		}
		s.Runtimes[v.Id] = proto.Clone(v).(*pb.Runtime)
		return nil
	})
	return &pb.RegisterRuntimeResponse{Runtime: proto.Clone(v).(*pb.Runtime)}, err
}
func (d *Hub) RegisterAgent(ctx context.Context, r *pb.RegisterAgentRequest) (*pb.RegisterAgentResponse, error) {
	a := r.Agent
	if a != nil && a.Disabled {
		return nil, invalid("disabled is managed by UnregisterAgent")
	}
	if a == nil || a.Id == "" || strings.TrimSpace(a.Name) == "" || len(a.Name) > 320 || len(a.Id) > 128 || len(r.RequestId) > 128 || r.DaemonId == "" || r.RequestId == "" {
		return nil, invalid("agent, name, daemon and request_id are required")
	}
	err := d.repo.update(ctx, func(s *localState) error {
		rt := s.Runtimes[a.RuntimeId]
		if rt == nil || rt.DaemonId != r.DaemonId {
			return conflict("runtime is not registered to this daemon")
		}
		if old := s.Agents[a.Id]; old != nil && old.RuntimeId != a.RuntimeId {
			return conflict("agent runtime cannot change")
		}
		s.Agents[a.Id] = proto.Clone(a).(*pb.Agent)
		return nil
	})
	return &pb.RegisterAgentResponse{Agent: proto.Clone(a).(*pb.Agent)}, err
}

// UnregisterAgent 仅停用注册，保留 Run、会话及 Runtime 的历史关联。
func (d *Hub) UnregisterAgent(ctx context.Context, r *pb.UnregisterAgentRequest) (*pb.UnregisterAgentResponse, error) {
	if r.RequestId == "" || r.DaemonId == "" || r.AgentId == "" || len(r.RequestId) > 128 || len(r.DaemonId) > 128 || len(r.AgentId) > 128 {
		return nil, invalid("request_id, daemon and agent are required")
	}
	err := d.repo.update(ctx, func(s *localState) error {
		key, body := "unregister:"+r.DaemonId+":"+r.RequestId, requestKey(r)
		if previous, ok := s.Requests[key]; ok {
			if previous.Body != body {
				return conflict("unregister request_id reused")
			}
			return nil
		}
		a := s.Agents[r.AgentId]
		if a != nil {
			if s.Runtimes[a.RuntimeId].DaemonId != r.DaemonId {
				return status.Error(codes.PermissionDenied, "agent belongs to another daemon")
			}
			for _, record := range s.Runs {
				if record.Run.AgentId == a.Id && !terminal(record.Run.State) {
					return conflict("agent has unfinished runs; finish or cancel them before unregistering")
				}
			}
			a.Disabled = true
		}
		s.Requests[key] = requestRecord{Body: body, IDs: []string{r.AgentId}}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.UnregisterAgentResponse{AgentId: r.AgentId}, nil
}

func (d *Hub) ListAgents(ctx context.Context, _ *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	out := &pb.ListAgentsResponse{}
	err := d.repo.view(ctx, func(s localState) error {
		for _, a := range s.Agents {
			rt := s.Runtimes[a.RuntimeId]
			out.Agents = append(out.Agents, &pb.AgentInfo{Agent: proto.Clone(a).(*pb.Agent), Runtime: proto.Clone(rt).(*pb.Runtime)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, a := range out.Agents {
		a.Online = !a.Agent.Disabled && d.online(a.Runtime.DaemonId)
	}
	sort.Slice(out.Agents, func(i, j int) bool { return out.Agents[i].Agent.Name < out.Agents[j].Agent.Name })
	return out, nil
}
func (d *Hub) CreateRun(ctx context.Context, r *pb.CreateRunRequest) (*pb.CreateRunResponse, error) {
	if r.RequestId == "" || r.AgentId == "" || strings.TrimSpace(r.Prompt) == "" || len(r.RequestId) > 128 || len(r.Prompt) > 48*1024 {
		return nil, invalid("request_id, agent and a prompt up to 48 KiB are required")
	}
	var result *pb.Run
	var daemon string
	err := d.repo.update(ctx, func(s *localState) error {
		key := "create:" + r.RequestId
		body := requestKey(r)
		if prev, ok := s.Requests[key]; ok {
			if prev.Body != body {
				return conflict("request_id reused with different input")
			}
			result = proto.Clone(s.Runs[prev.IDs[0]].Run).(*pb.Run)
			return nil
		}
		a := s.Agents[r.AgentId]
		if a == nil {
			return status.Error(codes.NotFound, "agent not found")
		}
		if a.Disabled {
			return conflict("agent is unregistered")
		}
		rt := s.Runtimes[a.RuntimeId]
		daemon = rt.DaemonId
		if !d.online(daemon) {
			return status.Error(codes.Unavailable, "daemon is offline")
		}
		session := r.SessionId
		if session == "" {
			session = uuid.NewString()
		}
		for _, record := range s.Runs {
			if record.Run.SessionId == session {
				if record.Run.AgentId != a.Id {
					return conflict("session belongs to another agent")
				}
				if !terminal(record.Run.State) {
					return conflict("session already has an active run")
				}
			}
		}
		if len(s.Runs) >= 512 {
			return status.Error(codes.ResourceExhausted, "Hub run history limit reached")
		}
		result = &pb.Run{Id: uuid.NewString(), AgentId: a.Id, RuntimeId: rt.Id, WorkspaceId: rt.WorkspaceId, SessionId: session, State: pb.RunState_RUN_STATE_QUEUED}
		s.Runs[result.Id] = &runRecord{Version: s.Revision, Run: proto.Clone(result).(*pb.Run), Prompt: r.Prompt, CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z07:00"), Commands: map[string]*pb.ControlCommand{}, Acks: map[string]*pb.CommandAck{}}
		s.Requests[key] = requestRecord{Body: body, IDs: []string{result.Id}}
		return nil
	})
	if err == nil {
		d.notify(daemon)
	}
	return &pb.CreateRunResponse{Run: result}, err
}
func (d *Hub) ClaimRuns(ctx context.Context, r *pb.ClaimRunsRequest) (*pb.ClaimRunsResponse, error) {
	if r.DaemonId == "" || r.RequestId == "" || r.Capacity > 4 || len(r.RequestId) > 128 {
		return nil, invalid("daemon, request_id and capacity <= 4 are required")
	}
	out := &pb.ClaimRunsResponse{}
	err := d.repo.update(ctx, func(s *localState) error {
		key := "claim:" + r.DaemonId + ":" + r.RequestId
		body := requestKey(r)
		ids := []string{}
		if prev, ok := s.Requests[key]; ok {
			if prev.Body != body {
				return conflict("claim request_id reused")
			}
			ids = prev.IDs
		} else {
			allowed := map[string]bool{}
			for _, id := range r.RuntimeIds {
				rt := s.Runtimes[id]
				if rt == nil || rt.DaemonId != r.DaemonId {
					return conflict("runtime does not belong to daemon")
				}
				allowed[id] = true
			}
			records := []*runRecord{}
			for _, record := range s.Runs {
				if allowed[record.Run.RuntimeId] && !s.Agents[record.Run.AgentId].Disabled && record.Run.State == pb.RunState_RUN_STATE_QUEUED {
					records = append(records, record)
				}
			}
			sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt < records[j].CreatedAt })
			busy := map[string]bool{}
			for _, record := range s.Runs {
				if record.Run.State != pb.RunState_RUN_STATE_QUEUED && !terminal(record.Run.State) {
					rt := s.Runtimes[record.Run.RuntimeId]
					if rt.DaemonId == r.DaemonId {
						busy[rt.WorkspaceId] = true
					}
				}
			}
			for _, record := range records {
				if len(ids) >= int(r.Capacity) {
					break
				}
				workspace := s.Runtimes[record.Run.RuntimeId].WorkspaceId
				if busy[workspace] {
					continue
				}
				busy[workspace] = true
				record.Run.State = pb.RunState_RUN_STATE_DISPATCHED
				record.Version = s.Revision
				record.Run.AttemptId = uuid.NewString()
				record.LeaseUntil = time.Now().Add(2 * time.Minute).Unix()
				ids = append(ids, record.Run.Id)
			}
			s.Requests[key] = requestRecord{Body: body, IDs: ids}
		}
		for _, id := range ids {
			record := s.Runs[id]
			out.Leases = append(out.Leases, &pb.RunLease{Run: proto.Clone(record.Run).(*pb.Run), Agent: proto.Clone(s.Agents[record.Run.AgentId]).(*pb.Agent), Prompt: record.Prompt, ExpiresAtUnixSeconds: record.LeaseUntil})
		}
		return nil
	})
	return out, err
}
func (d *Hub) GetRun(ctx context.Context, r *pb.GetRunRequest) (*pb.GetRunResponse, error) {
	var out *pb.Run
	err := d.repo.view(ctx, func(s localState) error {
		v := s.Runs[r.RunId]
		if v == nil {
			return status.Error(codes.NotFound, "run not found")
		}
		out = proto.Clone(v.Run).(*pb.Run)
		return nil
	})
	return &pb.GetRunResponse{Run: out}, err
}

func (d *Hub) control(ctx context.Context, c *pb.ControlCommand) error {
	if c.CommandId == "" || len(c.CommandId) > 128 || c.Attempt == nil || c.Attempt.RunId == "" {
		return invalid("command_id and attempt are required")
	}
	var daemon string
	err := d.repo.update(ctx, func(s *localState) error {
		rec := s.Runs[c.Attempt.RunId]
		if rec == nil {
			return status.Error(codes.NotFound, "run not found")
		}
		run := rec.Run
		daemon = s.Runtimes[run.RuntimeId].DaemonId
		if old := rec.Commands[c.CommandId]; old != nil {
			if !proto.Equal(old, c) {
				return conflict("command_id reused with different payload")
			}
			return nil
		}
		if run.AttemptId != c.Attempt.AttemptId || terminal(run.State) {
			return conflict("stale attempt or terminal run")
		}
		if len(rec.Commands) >= 256 {
			return status.Error(codes.ResourceExhausted, "too many commands")
		}
		switch p := c.Payload.(type) {
		case *pb.ControlCommand_Cancel:
			run.CancelRequested = true
			if run.State == pb.RunState_RUN_STATE_QUEUED {
				run.State = pb.RunState_RUN_STATE_CANCELLED
			}
		case *pb.ControlCommand_InteractionReply:
			if len(p.InteractionReply.ResponseJson) > 48*1024 {
				return invalid("interaction response exceeds 48 KiB")
			}
			var pending *pb.InteractionEvent
			for _, e := range rec.Events {
				if i := e.GetInteraction(); i != nil && i.InteractionId == p.InteractionReply.InteractionId {
					pending = i
				}
			}
			if pending == nil {
				return conflict("interaction does not belong to this attempt")
			}
			for id, old := range rec.Commands {
				if ack := rec.Acks[id]; ack != nil && ack.State == pb.CommandAckState_COMMAND_ACK_STATE_REJECTED {
					continue
				}
				if reply := old.GetInteractionReply(); reply != nil && reply.InteractionId == pending.InteractionId {
					return conflict("interaction already answered")
				}
			}
			var value map[string]any
			if json.Unmarshal(p.InteractionReply.ResponseJson, &value) != nil || value == nil {
				return invalid("interaction response must be a JSON object")
			}
			if run.State != pb.RunState_RUN_STATE_WAITING_INPUT && run.State != pb.RunState_RUN_STATE_WAITING_APPROVAL {
				return conflict("run is not waiting for a reply")
			}
		default:
			return status.Error(codes.Unimplemented, "operation is not supported by this Hub")
		}
		rec.Commands[c.CommandId] = proto.Clone(c).(*pb.ControlCommand)
		rec.Version = s.Revision
		return nil
	})
	if err == nil {
		d.notify(daemon)
	}
	return err
}
func (d *Hub) RequestCancel(ctx context.Context, r *pb.RequestCancelRequest) (*pb.RequestCancelResponse, error) {
	err := d.control(ctx, &pb.ControlCommand{CommandId: r.CommandId, Attempt: r.Attempt, Payload: &pb.ControlCommand_Cancel{Cancel: &pb.CancelCommand{Reason: r.Reason}}})
	return &pb.RequestCancelResponse{CommandId: r.CommandId}, err
}
func (d *Hub) ReplyInteraction(ctx context.Context, r *pb.ReplyInteractionRequest) (*pb.ReplyInteractionResponse, error) {
	err := d.control(ctx, &pb.ControlCommand{CommandId: r.CommandId, Attempt: r.Attempt, Payload: &pb.ControlCommand_InteractionReply{InteractionReply: &pb.InteractionReplyCommand{InteractionId: r.InteractionId, ResponseJson: r.ResponseJson}}})
	return &pb.ReplyInteractionResponse{CommandId: r.CommandId}, err
}
func (d *Hub) record(ctx context.Context, daemon string, e *pb.ExecutionEvent) (*pb.EventAck, error) {
	if e.Attempt == nil || e.Sequence == 0 || e.Payload == nil {
		return nil, invalid("event attempt, sequence and payload are required")
	}
	err := d.repo.update(ctx, func(s *localState) error {
		rec := s.Runs[e.Attempt.RunId]
		if rec == nil {
			return status.Error(codes.NotFound, "run not found")
		}
		run := rec.Run
		if s.Runtimes[run.RuntimeId].DaemonId != daemon || run.AttemptId != e.Attempt.AttemptId {
			return conflict("stale attempt or wrong daemon")
		}
		if e.Sequence <= uint64(len(rec.Events)) {
			if !proto.Equal(rec.Events[e.Sequence-1], e) {
				return conflict("event sequence reused with different content")
			}
			return nil
		}
		if e.Sequence != uint64(len(rec.Events))+1 || terminal(run.State) {
			return conflict("event gap or terminal run")
		}
		if v := e.GetState(); v != nil {
			if v.State != pb.RunState_RUN_STATE_RUNNING && v.State != pb.RunState_RUN_STATE_WAITING_INPUT && v.State != pb.RunState_RUN_STATE_WAITING_APPROVAL {
				return invalid("invalid execution state")
			}
			if run.State == pb.RunState_RUN_STATE_DISPATCHED && time.Now().Unix() > rec.LeaseUntil {
				return conflict("startup lease expired")
			}
			run.State = v.State
		}
		if v := e.GetResult(); v != nil {
			if !terminal(v.State) || (v.State == pb.RunState_RUN_STATE_CANCELLED && !v.ExecutionStopped) {
				return invalid("terminal result requires confirmed cancellation")
			}
			run.State = v.State
			run.Execution = v.Execution
		}
		rec.Events = append(rec.Events, proto.Clone(e).(*pb.ExecutionEvent))
		rec.EventRevisions = append(rec.EventRevisions, s.Revision)
		rec.Version = s.Revision
		return nil
	})
	return &pb.EventAck{Attempt: e.Attempt, ThroughSequence: e.Sequence}, err
}
func (d *Hub) acknowledge(ctx context.Context, daemon string, a *pb.CommandAck) error {
	return d.repo.update(ctx, func(s *localState) error {
		if a.Attempt == nil {
			return invalid("missing attempt")
		}
		r := s.Runs[a.Attempt.RunId]
		if r == nil || r.Run.AttemptId != a.Attempt.AttemptId || s.Runtimes[r.Run.RuntimeId].DaemonId != daemon || r.Commands[a.CommandId] == nil {
			return conflict("unknown command or wrong attempt")
		}
		if !proto.Equal(r.Acks[a.CommandId], a) {
			r.Acks[a.CommandId] = proto.Clone(a).(*pb.CommandAck)
			r.Version = s.Revision
		}
		return nil
	})
}
func (d *Hub) pending(ctx context.Context, daemon string) ([]*pb.ControlCommand, error) {
	var out []*pb.ControlCommand
	err := d.repo.view(ctx, func(s localState) error {
		for _, r := range s.Runs {
			if s.Runtimes[r.Run.RuntimeId].DaemonId == daemon && !terminal(r.Run.State) {
				for id, c := range r.Commands {
					if r.Acks[id] == nil {
						out = append(out, proto.Clone(c).(*pb.ControlCommand))
					}
				}
			}
		}
		return nil
	})
	return out, err
}
func (d *Hub) reconcile(ctx context.Context, daemon string) *pb.ReconcileResponse {
	out := &pb.ReconcileResponse{}
	_ = d.repo.view(ctx, func(s localState) error {
		for _, r := range s.Runs {
			if s.Runtimes[r.Run.RuntimeId].DaemonId == daemon && !terminal(r.Run.State) {
				out.Runs = append(out.Runs, proto.Clone(r.Run).(*pb.Run))
			}
			if s.Runtimes[r.Run.RuntimeId].DaemonId == daemon && len(r.Events) > 0 {
				out.EventAcks = append(out.EventAcks, &pb.EventAck{Attempt: &pb.AttemptRef{RunId: r.Run.Id, AttemptId: r.Run.AttemptId}, ThroughSequence: uint64(len(r.Events))})
			}
		}
		return nil
	})
	out.PendingCommands, _ = d.pending(ctx, daemon)
	return out
}
func (d *Hub) Connect(stream grpc.BidiStreamingServer[pb.ConnectRequest, pb.ConnectResponse]) error {
	ctx := stream.Context()
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	hello := first.GetHello()
	if hello == nil || hello.DaemonId == "" {
		return invalid("first frame must be Hello")
	}
	daemon := hello.DaemonId
	conn := &daemonConnection{wake: make(chan struct{}, 1), stop: make(chan struct{}), seen: time.Now()}
	d.mu.Lock()
	if old := d.connections[daemon]; old != nil {
		close(old.stop)
	}
	d.connections[daemon] = conn
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		if d.connections[daemon] == conn {
			delete(d.connections, daemon)
		}
		d.mu.Unlock()
	}()
	if err = stream.Send(&pb.ConnectResponse{Payload: &pb.ConnectResponse_Reconcile{Reconcile: d.reconcile(ctx, daemon)}}); err != nil {
		return err
	}
	frames := make(chan *pb.ConnectRequest)
	fail := make(chan error, 1)
	go func() {
		for {
			m, e := stream.Recv()
			if e != nil {
				fail <- e
				return
			}
			select {
			case frames <- m:
			case <-stream.Context().Done():
				return
			case <-conn.stop:
				return
			}
		}
	}()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	sendPending := func() error {
		commands, e := d.pending(ctx, daemon)
		if e != nil {
			return e
		}
		for _, c := range commands {
			if e = stream.Send(&pb.ConnectResponse{Payload: &pb.ConnectResponse_Command{Command: c}}); e != nil {
				return e
			}
		}
		return stream.Send(&pb.ConnectResponse{Payload: &pb.ConnectResponse_WorkAvailable{WorkAvailable: &pb.WorkAvailable{}}})
	}
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-conn.stop:
			return status.Error(codes.Aborted, "daemon stream replaced")
		case err = <-fail:
			return err
		case <-conn.wake:
			if err = sendPending(); err != nil {
				return err
			}
		case <-ticker.C:
			if err = sendPending(); err != nil {
				return err
			}
		case frame := <-frames:
			if hb := frame.GetHeartbeat(); hb != nil {
				d.mu.Lock()
				conn.seen = time.Now()
				d.mu.Unlock()
			}
			if e := frame.GetEvent(); e != nil {
				ack, e2 := d.record(ctx, daemon, e)
				if e2 != nil {
					return e2
				}
				if err = stream.Send(&pb.ConnectResponse{Payload: &pb.ConnectResponse_EventAck{EventAck: ack}}); err != nil {
					return err
				}
			}
			if ack := frame.GetCommandAck(); ack != nil {
				if err = d.acknowledge(ctx, daemon, ack); err != nil {
					return err
				}
			}
			if frame.GetReconcile() != nil {
				if err = stream.Send(&pb.ConnectResponse{Payload: &pb.ConnectResponse_Reconcile{Reconcile: d.reconcile(ctx, daemon)}}); err != nil {
					return err
				}
			}
		}
	}
}
