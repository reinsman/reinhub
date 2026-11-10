package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	pb "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"google.golang.org/protobuf/proto"
)

// shortcut: 当前使用单进程原子快照；支持多实例前改为事务数据库。
type localState struct {
	Epoch    string
	Revision uint64
	Runtimes map[string]*pb.Runtime
	Agents   map[string]*pb.Agent
	Runs     map[string]*runRecord
	Requests map[string]requestRecord
}
type requestRecord struct {
	Body string
	IDs  []string
}
type runRecord struct {
	Version        uint64
	EventRevisions []uint64
	Run            *pb.Run `json:"run"`
	Prompt         string  `json:"prompt"`
	CreatedAt      string  `json:"created_at"`
	LeaseUntil     int64
	Events         []*pb.ExecutionEvent
	Commands       map[string]*pb.ControlCommand
	Acks           map[string]*pb.CommandAck
}
type localRepository struct {
	mu      sync.Mutex
	path    string
	state   localState
	failed  error
	changed chan struct{}
}

func openLocalRepository(path string) (*localRepository, error) {
	r := &localRepository{path: path, changed: make(chan struct{}), state: localState{Epoch: uuid.NewString(), Runtimes: map[string]*pb.Runtime{}, Agents: map[string]*pb.Agent{}, Runs: map[string]*runRecord{}, Requests: map[string]requestRecord{}}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &r.state); err != nil {
		return nil, fmt.Errorf("read Hub state: %w", err)
	}
	if r.state.Runtimes == nil || r.state.Agents == nil || r.state.Runs == nil || r.state.Requests == nil {
		return nil, fmt.Errorf("invalid Hub state")
	}
	if r.state.Epoch == "" {
		r.state.Epoch = uuid.NewString()
	}
	for id, agent := range r.state.Agents {
		if agent == nil || agent.Id != id || r.state.Runtimes[agent.RuntimeId] == nil {
			return nil, fmt.Errorf("invalid persisted Agent/runtime mapping")
		}
	}
	for id, record := range r.state.Runs {
		if record == nil || record.Run == nil || record.Run.Id != id || r.state.Agents[record.Run.AgentId] == nil || r.state.Runtimes[record.Run.RuntimeId] == nil {
			return nil, fmt.Errorf("invalid persisted Run mapping")
		}
	}
	return r, nil
}
func (r *localRepository) update(ctx context.Context, fn func(*localState) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.failed != nil {
		return fmt.Errorf("Hub storage requires restart after uncertain commit: %w", r.failed)
	}
	data, err := json.Marshal(r.state)
	if err != nil {
		return err
	}
	var next localState
	if err = json.Unmarshal(data, &next); err != nil {
		return err
	}
	next.Revision++
	if err = fn(&next); err != nil {
		return err
	}
	data, err = json.Marshal(next)
	if err != nil {
		return err
	}
	if len(data) > 64*1024*1024 {
		return fmt.Errorf("Hub history exceeds 64 MiB")
	}
	if err = os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(r.path), ".hub-state-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(file.Name(), r.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(r.path))
	if err == nil {
		err = dir.Sync()
		_ = dir.Close()
	}
	if err != nil {
		r.failed = err
		return err
	}
	r.state = next
	close(r.changed)
	r.changed = make(chan struct{})
	return nil
}
func (r *localRepository) view(ctx context.Context, fn func(localState) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn(r.state)
}

// protobuf oneof 不能交给 encoding/json 直接反序列化；快照中保存生成类型的 wire bytes。
type wireRunRecord struct {
	Version        uint64
	EventRevisions []uint64
	Run            []byte
	Prompt         string
	CreatedAt      string
	LeaseUntil     int64
	Events         [][]byte
	Commands       map[string][]byte
	Acks           map[string][]byte
}

func (r *runRecord) MarshalJSON() ([]byte, error) {
	wire := wireRunRecord{Version: r.Version, EventRevisions: r.EventRevisions, Prompt: r.Prompt, CreatedAt: r.CreatedAt, LeaseUntil: r.LeaseUntil, Commands: map[string][]byte{}, Acks: map[string][]byte{}}
	var err error
	if wire.Run, err = proto.Marshal(r.Run); err != nil {
		return nil, err
	}
	for _, event := range r.Events {
		data, e := proto.Marshal(event)
		if e != nil {
			return nil, e
		}
		wire.Events = append(wire.Events, data)
	}
	for id, command := range r.Commands {
		data, e := proto.Marshal(command)
		if e != nil {
			return nil, e
		}
		wire.Commands[id] = data
	}
	for id, ack := range r.Acks {
		data, e := proto.Marshal(ack)
		if e != nil {
			return nil, e
		}
		wire.Acks[id] = data
	}
	return json.Marshal(wire)
}
func (r *runRecord) UnmarshalJSON(data []byte) error {
	var wire wireRunRecord
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	r.Run = &pb.Run{}
	if err := proto.Unmarshal(wire.Run, r.Run); err != nil {
		return err
	}
	r.Prompt = wire.Prompt
	r.Version = wire.Version
	r.EventRevisions = wire.EventRevisions
	r.CreatedAt = wire.CreatedAt
	r.LeaseUntil = wire.LeaseUntil
	r.Events = nil
	r.Commands = map[string]*pb.ControlCommand{}
	r.Acks = map[string]*pb.CommandAck{}
	for _, data := range wire.Events {
		event := &pb.ExecutionEvent{}
		if err := proto.Unmarshal(data, event); err != nil {
			return err
		}
		r.Events = append(r.Events, event)
	}
	// 旧快照没有提交游标；既有事件属于初始 HTTP 历史，不作为新 SSE 增量。
	if len(r.EventRevisions) == 0 {
		r.EventRevisions = make([]uint64, len(r.Events))
	}
	if len(r.EventRevisions) != len(r.Events) {
		return fmt.Errorf("invalid event revision count")
	}
	for id, data := range wire.Commands {
		command := &pb.ControlCommand{}
		if err := proto.Unmarshal(data, command); err != nil {
			return err
		}
		r.Commands[id] = command
	}
	for id, data := range wire.Acks {
		ack := &pb.CommandAck{}
		if err := proto.Unmarshal(data, ack); err != nil {
			return err
		}
		r.Acks[id] = ack
	}
	return nil
}
