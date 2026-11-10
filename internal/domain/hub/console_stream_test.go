package hub

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	pb "github.com/reinsman/proto/gen/go/rein/hub/v1"
	api "github.com/reinsman/reinhub/pkg/api/v1/hub"
)

func TestHistoryIsScopedAndStableWhileNewRunsArrive(t *testing.T) {
	d, _ := setupHub(t)
	ctx := context.Background()
	insert := func(start, end int) {
		t.Helper()
		if err := d.repo.update(ctx, func(s *localState) error {
			for i := start; i <= end; i++ {
				id := fmt.Sprintf("run-%03d", i)
				s.Runs[id] = &runRecord{Version: s.Revision, Run: &pb.Run{Id: id, AgentId: "agent", RuntimeId: "runtime", SessionId: "session", State: pb.RunState_RUN_STATE_COMPLETED}, CreatedAt: "2026-01-01T00:00:00Z", Prompt: id, Commands: map[string]*pb.ControlCommand{}, Acks: map[string]*pb.CommandAck{}}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	insert(1, 25)
	page, err := d.history(ctx, "agent", "session", "", 20)
	if err != nil || len(page.Runs) != 20 || page.Runs[0].Prompt != "run-006" || page.NextCursor == "" {
		t.Fatalf("first page: %+v %v", page, err)
	}
	insert(26, 26)
	older, err := d.history(ctx, "agent", "session", page.NextCursor, 20)
	if err != nil || len(older.Runs) != 5 || older.Runs[0].Prompt != "run-001" || older.NextCursor != "" {
		t.Fatalf("older page: %+v %v", older, err)
	}
	if _, err = d.history(ctx, "agent", "other", page.NextCursor, 20); err == nil {
		t.Fatal("cross-session cursor accepted")
	}
	if _, err = d.history(ctx, "agent", "session", "bad", 20); err == nil {
		t.Fatal("invalid cursor accepted")
	}
	if _, err = d.history(ctx, "agent", "session", "", 51); err == nil {
		t.Fatal("unbounded page accepted")
	}
}

type sseFrame struct{ kind, id, data string }

func readFrame(t *testing.T, reader *bufio.Reader) sseFrame {
	t.Helper()
	var frame sseFrame
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			if frame.kind != "" {
				return frame
			}
			continue
		}
		key, value, _ := strings.Cut(line, ": ")
		switch key {
		case "event":
			frame.kind = value
		case "id":
			frame.id = value
		case "data":
			frame.data = value
		}
	}
}

func TestSSEReplaysDurableChangesAfterHTTPAndCheckpointSurvivesRestart(t *testing.T) {
	d, path := setupHub(t)
	ctx := context.Background()
	run := createAndClaim(t, d)
	a := &pb.AttemptRef{RunId: run.Id, AttemptId: run.AttemptId}
	record := func(seq uint64, text string) {
		t.Helper()
		if _, err := d.record(ctx, "daemon", &pb.ExecutionEvent{Attempt: a, Sequence: seq, Payload: &pb.ExecutionEvent_Text{Text: &pb.TextEvent{Text: text}}}); err != nil {
			t.Fatal(err)
		}
	}
	record(1, "before HTTP")
	page, err := d.history(ctx, "agent", run.SessionId, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	record(2, "between HTTP and SSE")
	server := httptest.NewUnstartedServer(d.ConsoleHandler())
	server.Config.WriteTimeout = 10 * time.Millisecond
	server.Start()
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	endpoint := server.URL + "/api/agents/agent/events?cursor=" + url.QueryEscape(page.StreamCursor)
	response, err := client.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal(response.Status)
	}
	reader := bufio.NewReader(response.Body)
	frame := readFrame(t, reader)
	var update api.RunUpdate
	if frame.kind != "update" || json.Unmarshal([]byte(frame.data), &update) != nil || len(update.Events) != 1 || !strings.Contains(string(update.Events[0]), "between HTTP and SSE") {
		t.Fatalf("unexpected replay: %+v", frame)
	}
	checkpoint := readFrame(t, reader)
	if checkpoint.kind != "checkpoint" || checkpoint.id == "" {
		t.Fatal(checkpoint)
	}
	// SSE 单独续写超时，不受普通 HTTP 的短 WriteTimeout 截断。
	time.Sleep(30 * time.Millisecond)
	record(3, "live update")
	frame = readFrame(t, reader)
	if !strings.Contains(frame.data, "live update") {
		t.Fatal(frame)
	}
	checkpoint = readFrame(t, reader)
	response.Body.Close()
	record(4, "after disconnect")
	reopened, err := NewHub(path)
	if err != nil {
		t.Fatal(err)
	}
	epoch, number, _ := strings.Cut(checkpoint.id, ":")
	after, _ := strconv.ParseUint(number, 10, 64)
	updates, _, _, reset, err := reopened.streamUpdates(ctx, "agent", epoch, after)
	if err != nil || reset || len(updates) != 1 || len(updates[0].Events) != 1 || !strings.Contains(string(updates[0].Events[0]), "after disconnect") {
		t.Fatalf("restart replay: %v %v %v", updates, reset, err)
	}
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Last-Event-ID", checkpoint.id)
	resumed, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Body.Close()
	frame = readFrame(t, bufio.NewReader(resumed.Body))
	if strings.Contains(frame.data, "between HTTP and SSE") || !strings.Contains(frame.data, "after disconnect") {
		t.Fatalf("Last-Event-ID ignored: %+v", frame)
	}
	resetResponse, err := client.Get(server.URL + "/api/agents/agent/events?cursor=old-epoch:0")
	if err != nil {
		t.Fatal(err)
	}
	defer resetResponse.Body.Close()
	if readFrame(t, bufio.NewReader(resetResponse.Body)).kind != "reset" {
		t.Fatal("reset was not delivered")
	}
}

func TestLegacySnapshotGetsSafeStreamRevisions(t *testing.T) {
	d, path := setupHub(t)
	run := createAndClaim(t, d)
	ctx := context.Background()
	event := &pb.ExecutionEvent{Attempt: &pb.AttemptRef{RunId: run.Id, AttemptId: run.AttemptId}, Sequence: 1, Payload: &pb.ExecutionEvent_State{State: &pb.StateEvent{State: pb.RunState_RUN_STATE_RUNNING}}}
	if _, err := d.record(ctx, "daemon", event); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var state map[string]json.RawMessage
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	delete(state, "Epoch")
	delete(state, "Revision")
	var runs map[string]map[string]json.RawMessage
	if err := json.Unmarshal(state["Runs"], &runs); err != nil {
		t.Fatal(err)
	}
	delete(runs[run.Id], "Version")
	delete(runs[run.Id], "EventRevisions")
	state["Runs"], _ = json.Marshal(runs)
	data, _ = json.Marshal(state)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewHub(path)
	if err != nil {
		t.Fatal(err)
	}
	page, err := reopened.history(ctx, "agent", run.SessionId, "", 20)
	if err != nil || len(page.Runs[0].Events) != 1 {
		t.Fatalf("old history lost: %v", err)
	}
	event.Sequence = 2
	event.Payload = &pb.ExecutionEvent_Text{Text: &pb.TextEvent{Text: "new"}}
	if _, err = reopened.record(ctx, "daemon", event); err != nil {
		t.Fatal(err)
	}
	epoch, _, _ := strings.Cut(page.StreamCursor, ":")
	updates, _, _, _, err := reopened.streamUpdates(ctx, "agent", epoch, 0)
	if err != nil || len(updates) != 1 || len(updates[0].Events) != 1 {
		t.Fatalf("legacy replay: %v %v", updates, err)
	}
}

func TestSSEDrainsMultipleCommitBatchesWithoutSkippingAnotherRun(t *testing.T) {
	d, _ := setupHub(t)
	ctx := context.Background()
	if err := d.repo.update(ctx, func(s *localState) error {
		for _, id := range []string{"a", "b"} {
			s.Runs[id] = &runRecord{Version: s.Revision, Run: &pb.Run{Id: id, AttemptId: id, AgentId: "agent", RuntimeId: "runtime", SessionId: id, State: pb.RunState_RUN_STATE_RUNNING}, Commands: map[string]*pb.ControlCommand{}, Acks: map[string]*pb.CommandAck{}}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	page, err := d.history(ctx, "agent", "a", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 80; i++ {
		id := []string{"a", "b"}[i%2]
		if _, err = d.record(ctx, "daemon", &pb.ExecutionEvent{Attempt: &pb.AttemptRef{RunId: id, AttemptId: id}, Sequence: uint64(i/2 + 1), Payload: &pb.ExecutionEvent_Text{Text: &pb.TextEvent{Text: strconv.Itoa(i)}}}); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(d.ConsoleHandler())
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(server.URL + "/api/agents/agent/events?cursor=" + url.QueryEscape(page.StreamCursor))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	seen := map[string]bool{}
	for checkpoints := 0; checkpoints < 2; {
		frame := readFrame(t, reader)
		if frame.kind == "checkpoint" {
			checkpoints++
			continue
		}
		var update api.RunUpdate
		if json.Unmarshal([]byte(frame.data), &update) != nil {
			t.Fatal(frame)
		}
		for _, data := range update.Events {
			var event struct {
				Text struct {
					Text string `json:"text"`
				} `json:"text"`
			}
			if json.Unmarshal(data, &event) != nil || seen[event.Text.Text] {
				t.Fatal("event duplicated")
			}
			seen[event.Text.Text] = true
		}
	}
	if len(seen) != 80 {
		t.Fatalf("replay lost events: %d", len(seen))
	}
}
