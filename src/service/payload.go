package service

type Payload struct {
	Type    string      `json:"eventType"`
	Tick    int64       `json:"tick"`
	MatchID string      `json:"matchId"`
	Data    interface{} `json:"data"`
}
