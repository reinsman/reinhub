package ping

type Pong struct {
	CommitId string `json:"commitid"`
	Health   int    `json:"health"`
}
