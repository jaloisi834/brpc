package service

import (
	uuid "github.com/satori/go.uuid"
)

// player speed (maybe rename?)
const speed = 10

type Actor struct {
	ID        string `json:"id"`
	IGN       string `json:"ign"`
	Eating    int    `json:"eating"` // countdown for the powered up state
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction []int  `json:"direction"`
	SkinID    string `json:"skinId"`
}

func NewActor(ign string, x, y int) *Actor {
	return &Actor{
		ID:        uuid.Must(uuid.NewV4()).String(),
		IGN:       ign,
		X:         x,
		Y:         y,
		Direction: make([]int, 2),
	}
}

func (a *Actor) move(distance []int) {
	a.X += distance[0]
	a.Y += distance[1]
}

// returns the bottom right corner of the player
func (a *Actor) getPoint2() (int, int) {
	return a.X + (gridSize * 2),
		a.Y + (gridSize * 2)
}
