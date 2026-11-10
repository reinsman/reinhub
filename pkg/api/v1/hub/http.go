package hub

import "encoding/json"

// REST 只承载 Console 操作；嵌套协议消息使用唯一 proto 源的 JSON 表示。
type RunView struct {
	Version     string            `json:"version"`
	Run         json.RawMessage   `json:"run"`
	Prompt      string            `json:"prompt"`
	CreatedAt   string            `json:"created_at"`
	CommandAcks []json.RawMessage `json:"command_acks"`
}

type RunUpdate struct {
	RunView
	Events []json.RawMessage `json:"events"`
}

type SessionView struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

type HistoryPage struct {
	Runs         []RunUpdate   `json:"runs"`
	Sessions     []SessionView `json:"sessions"`
	SessionID    string        `json:"session_id"`
	NextCursor   string        `json:"next_cursor"`
	StreamCursor string        `json:"stream_cursor"`
}
type CreateRunRequest struct {
	RequestID string `json:"request_id"`
	AgentID   string `json:"agent_id"`
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
}
type ControlRequest struct {
	CommandID     string          `json:"command_id"`
	AttemptID     string          `json:"attempt_id"`
	InteractionID string          `json:"interaction_id"`
	Response      json.RawMessage `json:"response"`
}
