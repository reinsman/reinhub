package hub

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	pb "github.com/reinsman/proto/gen/go/rein/hub/v1"
	api "github.com/reinsman/reinhub/pkg/api/v1/hub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func protocolJSON(message proto.Message) json.RawMessage {
	b, _ := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(message)
	return b
}
func respond(w http.ResponseWriter, value any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		code := http.StatusInternalServerError
		switch status.Code(err) {
		case codes.InvalidArgument:
			code = 400
		case codes.NotFound:
			code = 404
		case codes.FailedPrecondition, codes.AlreadyExists:
			code = 409
		case codes.Unimplemented:
			code = 501
		case codes.Unavailable:
			code = 503
		case codes.ResourceExhausted:
			code = 429
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": status.Convert(err).Message()})
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}
func decode(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return invalid("invalid request JSON")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return invalid("request must contain exactly one JSON value")
	}
	return nil
}
func (d *Hub) ConsoleHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/agents/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		if value := r.URL.Query().Get("limit"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				respond(w, nil, invalid("invalid history limit"))
				return
			}
			limit = parsed
		}
		page, err := d.history(r.Context(), r.PathValue("id"), r.URL.Query().Get("session_id"), r.URL.Query().Get("before"), limit)
		respond(w, page, err)
	})
	mux.HandleFunc("GET /api/agents/{id}/events", d.streamRuns)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]string{"status": "ok"}, nil)
	})
	mux.HandleFunc("GET /api/agents", func(w http.ResponseWriter, r *http.Request) {
		agents, err := d.ListAgents(r.Context(), &pb.ListAgentsRequest{})
		if err != nil {
			respond(w, nil, err)
			return
		}
		respond(w, protocolJSON(agents), nil)
	})
	mux.HandleFunc("POST /api/runs", func(w http.ResponseWriter, r *http.Request) {
		var req api.CreateRunRequest
		if err := decode(w, r, &req); err != nil {
			respond(w, nil, err)
			return
		}
		res, err := d.CreateRun(r.Context(), &pb.CreateRunRequest{RequestId: req.RequestID, AgentId: req.AgentID, SessionId: req.SessionID, Prompt: req.Prompt})
		if err != nil {
			respond(w, nil, err)
			return
		}
		respond(w, protocolJSON(res), nil)
	})
	mux.HandleFunc("GET /api/runs", func(w http.ResponseWriter, r *http.Request) {
		views := []api.RunView{}
		err := d.repo.view(r.Context(), func(s localState) error {
			for _, rec := range s.Runs {
				if rec.Run.AgentId == r.URL.Query().Get("agent_id") {
					views = append(views, runView(rec))
				}
			}
			return nil
		})
		sort.Slice(views, func(i, j int) bool { return views[i].CreatedAt < views[j].CreatedAt })
		respond(w, views, err)
	})
	mux.HandleFunc("GET /api/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		res, err := d.GetRun(r.Context(), &pb.GetRunRequest{RunId: r.PathValue("id")})
		if err != nil {
			respond(w, nil, err)
			return
		}
		respond(w, protocolJSON(res), nil)
	})
	mux.HandleFunc("GET /api/runs/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		after, err := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
		if err != nil {
			respond(w, nil, invalid("after must be an unsigned sequence"))
			return
		}
		events := []json.RawMessage{}
		err = d.repo.view(r.Context(), func(s localState) error {
			rec := s.Runs[r.PathValue("id")]
			if rec == nil {
				return status.Error(codes.NotFound, "run not found")
			}
			if rec.Run.AttemptId != r.URL.Query().Get("attempt_id") {
				return conflict("attempt mismatch")
			}
			bytes := 0
			for _, e := range rec.Events {
				if e.Sequence > after {
					data := protocolJSON(e)
					if len(events) > 0 && (len(events) >= 100 || bytes+len(data) > 8*1024*1024) {
						break
					}
					events = append(events, data)
					bytes += len(data)
				}
			}
			return nil
		})
		respond(w, events, err)
	})
	mux.HandleFunc("POST /api/runs/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		var req api.ControlRequest
		if err := decode(w, r, &req); err != nil {
			respond(w, nil, err)
			return
		}
		res, err := d.RequestCancel(r.Context(), &pb.RequestCancelRequest{Attempt: &pb.AttemptRef{RunId: r.PathValue("id"), AttemptId: req.AttemptID}, CommandId: req.CommandID, Reason: "Console"})
		if err != nil {
			respond(w, nil, err)
			return
		}
		respond(w, protocolJSON(res), nil)
	})
	mux.HandleFunc("POST /api/runs/{id}/reply", func(w http.ResponseWriter, r *http.Request) {
		var req api.ControlRequest
		if err := decode(w, r, &req); err != nil {
			respond(w, nil, err)
			return
		}
		res, err := d.ReplyInteraction(r.Context(), &pb.ReplyInteractionRequest{Attempt: &pb.AttemptRef{RunId: r.PathValue("id"), AttemptId: req.AttemptID}, CommandId: req.CommandID, InteractionId: req.InteractionID, ResponseJson: req.Response})
		if err != nil {
			respond(w, nil, err)
			return
		}
		respond(w, protocolJSON(res), nil)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 当前无登录 API 可驱动本机进程，拒绝 DNS rebinding 和跨站请求。
		host := r.Host
		if parsed, err := url.Parse("http://" + host); err != nil || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "[::1]" && parsed.Hostname() != "::1") {
			http.Error(w, "loopback Host required", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Scheme != "http" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
				http.Error(w, "local Console origin required", 403)
				return
			}
		}
		if r.Method == http.MethodPost && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "application/json required", 415)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
