package hub

import (
	hubv1 "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"time"
)

// 权威状态模型骨架；只有事务语义实现并审查后才接入模板的 AutoMigrate。
type Agent struct {
	ID            string `gorm:"primaryKey;size:64"`
	WorkspaceID   string `gorm:"index;size:64"`
	RuntimeID     string `gorm:"size:64"`
	Configuration []byte
}

type Runtime struct {
	ID           string `gorm:"primaryKey;size:64"`
	WorkspaceID  string `gorm:"index;size:64"`
	DaemonID     string `gorm:"index;size:64"`
	OwnerID      string `gorm:"size:64"`
	Backend      hubv1.BackendKind
	Capabilities []byte
	LastSeenAt   time.Time
}

type Session struct {
	ID      string `gorm:"primaryKey;size:64"`
	AgentID string `gorm:"index;size:64"`
}

type Run struct {
	ID              string `gorm:"primaryKey;size:64"`
	SessionID       string `gorm:"index;size:64"`
	RuntimeID       string `gorm:"index;size:64"`
	AttemptID       string `gorm:"size:64"`
	State           hubv1.RunState
	CancelRequested bool
}

type Attempt struct {
	ID             string `gorm:"primaryKey;size:64"`
	RunID          string `gorm:"index;size:64"`
	LeaseExpiresAt time.Time
	AckedSequence  uint64
}

type ExecutionEvent struct {
	RunID     string `gorm:"primaryKey;size:64"`
	AttemptID string `gorm:"primaryKey;size:64"`
	Sequence  uint64 `gorm:"primaryKey"`
	Payload   []byte
}

type ControlCommand struct {
	ID        string `gorm:"primaryKey;size:64"`
	RunID     string `gorm:"index;size:64"`
	AttemptID string `gorm:"size:64"`
	Payload   []byte
	AckState  hubv1.CommandAckState
}
