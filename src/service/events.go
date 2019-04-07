package service

type DirectionEvent struct {
	MatchID      string `json:"matchId"`
	PlayerID     string `json:"playerId"`
	NewDirection []int  `json:"newDirection"`
}

type DeathEvent struct {
	MatchID string
	Tick    int64
	killer  Actor
	victim  Actor
}
