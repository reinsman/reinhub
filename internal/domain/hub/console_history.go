package hub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strconv"

	api "github.com/reinsman/reinhub/pkg/api/v1/hub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type historyCursor struct{ Agent, Session, Created, ID string }

func runView(rec *runRecord) api.RunView {
	acks := []json.RawMessage{}
	for _, ack := range rec.Acks {
		acks = append(acks, protocolJSON(ack))
	}
	return api.RunView{Version: strconv.FormatUint(rec.Version, 10), Run: protocolJSON(rec.Run), Prompt: rec.Prompt, CreatedAt: rec.CreatedAt, CommandAcks: acks}
}

// history 使用稳定的时间和 ID 游标；新任务插入不会改变历史分页边界。
func (d *Hub) history(ctx context.Context, agent, session, before string, limit int) (*api.HistoryPage, error) {
	if limit < 1 || limit > 50 {
		return nil, invalid("limit must be between 1 and 50")
	}
	var cursor historyCursor
	if before != "" {
		data, err := base64.RawURLEncoding.DecodeString(before)
		if len(before) > 1024 || err != nil || json.Unmarshal(data, &cursor) != nil || cursor.Agent != agent || cursor.Session != session || cursor.Created == "" || cursor.ID == "" {
			return nil, invalid("invalid history cursor")
		}
	}
	out := &api.HistoryPage{Runs: []api.RunUpdate{}, Sessions: []api.SessionView{}}
	err := d.repo.view(ctx, func(s localState) error {
		if s.Agents[agent] == nil {
			return status.Error(codes.NotFound, "agent not found")
		}
		out.StreamCursor = s.Epoch + ":" + strconv.FormatUint(s.Revision, 10)
		records := []*runRecord{}
		for _, rec := range s.Runs {
			if rec.Run.AgentId == agent {
				records = append(records, rec)
			}
		}
		sort.Slice(records, func(i, j int) bool {
			if records[i].CreatedAt == records[j].CreatedAt {
				return records[i].Run.Id > records[j].Run.Id
			}
			return records[i].CreatedAt > records[j].CreatedAt
		})
		seen := map[string]bool{}
		for _, rec := range records {
			if !seen[rec.Run.SessionId] {
				seen[rec.Run.SessionId] = true
				out.Sessions = append(out.Sessions, api.SessionView{ID: rec.Run.SessionId, Prompt: rec.Prompt})
			}
		}
		if before == "" && !seen[session] && len(out.Sessions) > 0 {
			session = out.Sessions[0].ID
		}
		out.SessionID = session
		page := []*runRecord{}
		for _, rec := range records {
			if rec.Run.SessionId != session {
				continue
			}
			if before != "" && (rec.CreatedAt > cursor.Created || (rec.CreatedAt == cursor.Created && rec.Run.Id >= cursor.ID)) {
				continue
			}
			page = append(page, rec)
		}
		if len(page) > limit {
			page = page[:limit]
			last := page[len(page)-1]
			data, _ := json.Marshal(historyCursor{Agent: agent, Session: session, Created: last.CreatedAt, ID: last.Run.Id})
			out.NextCursor = base64.RawURLEncoding.EncodeToString(data)
		}
		for i := len(page) - 1; i >= 0; i-- {
			rec := page[i]
			item := api.RunUpdate{RunView: runView(rec), Events: []json.RawMessage{}}
			for _, event := range rec.Events {
				item.Events = append(item.Events, protocolJSON(event))
			}
			out.Runs = append(out.Runs, item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
