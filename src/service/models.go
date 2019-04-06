package service

import uuid "github.com/satori/go.uuid"

// TODO: Make dynamic?
const gridSize = 23

type DirectionEvent struct {
	MatchID      string `json:"matchId"`
	Tick         int64  `json:"tick"`
	actor        Actor  `json:"actor"`
	NewDirection []int  `json:"newDirection"`
}

type DeathEvent struct {
	MatchID string
	Tick    int64
	killer  Actor
	victim  Actor
}

type Actor struct {
	ID        string `json:"id"`
	Eating    int    `json:"eating"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction []int  `json"direction"`
}

func (a *Actor) setPosition(x, y int) {
	a.X = x
	a.Y = y
}

type Match struct {
	ID       string
	GridSize int
	Map      [][]bool
	Players  map[string]*Actor // [playerID]Actor
}

func NewMatch() *Match {
	//TODO: build map
	return &Match{
		GridSize: gridSize,
		ID:       uuid.Must(uuid.NewV4()).String(),
		Players:  make(map[string]*Actor),
	}
}
