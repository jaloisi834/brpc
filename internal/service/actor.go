package service

import (
	uuid "github.com/satori/go.uuid"
)

// player speed (maybe rename?)
const speed = 5.0

type Actor struct {
	ID        string  `json:"id"`
	IGN       string  `json:"ign"`
	Eating    int     `json:"eating"` // countdown for the powered up state
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Direction []int   `json:"direction"`
	SkinID    string  `json:"skinId"`
	Dead      bool    `json:"dead"`
}

func NewActor(ign string, x, y float32) *Actor {
	return &Actor{
		ID:        uuid.NewV4().String(),
		IGN:       ign,
		X:         x,
		Y:         y,
		Direction: make([]int, 2),
	}
}

func (a *Actor) move(distance []float32) {
	a.X += distance[0]
	a.Y += distance[1]
}

// returns the bottom right corner of the player
func (a *Actor) getPoint2() (float32, float32) {
	return a.X + (gridSize * 2),
		a.Y + (gridSize * 2)
}

func (a *Actor) reverseDirection() {
	a.Direction[0] *= -1
	a.Direction[1] *= -1
}

func (a *Actor) processDamage(damage int) {
	a.Eating -= damage

	// If their health reaches or goes below 0, they died
	if a.Eating <= 0 {
		a.Eating = 0
		a.Dead = true
	}
}
