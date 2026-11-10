package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	api "github.com/reinsman/reinhub/pkg/api/v1/hub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// streamUpdates 读取持久化后的增量，并在同一把锁下取得下一次提交通知，避免丢失唤醒。
func (d *Hub) streamUpdates(ctx context.Context, agent, epoch string, after uint64) ([]api.RunUpdate, uint64, <-chan struct{}, bool, error) {
	updates := []api.RunUpdate{}
	var through uint64
	var changed <-chan struct{}
	reset := false
	err := d.repo.view(ctx, func(s localState) error {
		if s.Agents[agent] == nil {
			return status.Error(codes.NotFound, "agent not found")
		}
		changed = d.repo.changed
		if epoch != s.Epoch || after > s.Revision {
			reset = true
			return nil
		}
		through = s.Revision
		if through-after > 64 {
			through = after + 64
		}
		for _, rec := range s.Runs {
			if rec.Run.AgentId != agent || rec.Version <= after {
				continue
			}
			item := api.RunUpdate{RunView: runView(rec), Events: []json.RawMessage{}}
			for i, event := range rec.Events {
				if rec.EventRevisions[i] > after && rec.EventRevisions[i] <= through {
					item.Events = append(item.Events, protocolJSON(event))
				}
			}
			updates = append(updates, item)
		}
		return nil
	})
	return updates, through, changed, reset, err
}

func (d *Hub) streamRuns(w http.ResponseWriter, r *http.Request) {
	cursor := r.Header.Get("Last-Event-ID")
	if cursor == "" {
		cursor = r.URL.Query().Get("cursor")
	}
	epoch, number, ok := strings.Cut(cursor, ":")
	after, err := strconv.ParseUint(number, 10, 64)
	if !ok || len(epoch) > 128 || epoch == "" || err != nil {
		respond(w, nil, invalid("a valid stream cursor is required"))
		return
	}
	controller := http.NewResponseController(w)
	write := func(event string, value any, id string) error {
		data, e := json.Marshal(value)
		if e != nil {
			return e
		}
		if e = controller.SetWriteDeadline(time.Now().Add(10 * time.Second)); e != nil {
			return e
		}
		if id != "" {
			if _, e = fmt.Fprintf(w, "id: %s\n", id); e != nil {
				return e
			}
		}
		if _, e = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); e != nil {
			return e
		}
		return controller.Flush()
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	first := true
	for {
		updates, through, changed, reset, e := d.streamUpdates(r.Context(), r.PathValue("id"), epoch, after)
		if e != nil {
			if first {
				respond(w, nil, e)
			}
			return
		}
		if first {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("X-Accel-Buffering", "no")
		}
		if reset {
			_ = write("reset", struct{}{}, "")
			return
		}
		for _, update := range updates {
			if e = write("update", update, ""); e != nil {
				return
			}
		}
		// 只在完整批次之后推进 Last-Event-ID；中途断线会重放并由客户端去重。
		if first || through != after {
			if e = write("checkpoint", struct{}{}, epoch+":"+strconv.FormatUint(through, 10)); e != nil {
				return
			}
		}
		first = false
		previous := after
		after = through
		if through-previous == 64 {
			continue
		}
		select {
		case <-r.Context().Done():
			return
		case <-changed:
		case <-ticker.C:
			if err = controller.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if _, err = fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if err = controller.Flush(); err != nil {
				return
			}
		}
	}
}
