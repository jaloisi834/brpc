package service

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

const minPlayers = 2

const keyEventType = "type"

const (
	eventTypeDirection = "direction"
)

type Service struct {
	mutex          sync.Mutex // Protect matches access
	matches        map[string]*Match
	CurrentMatchID string // TODO: Don't use the concept of current match
}

func New() *Service {
	return &Service{
		matches: make(map[string]*Match, 1),
	}
}

// TODO: Maybe return a more specific type here?
func (s *Service) ProcessEvent(msgBytes []byte) (interface{}, error) {
	msg := string(msgBytes)
	log.Info().Str("message", msg).Msg("Received client message")

	// Take action based on the event type
	eventType := gjson.Get(msg, keyEventType).String()
	switch eventType {
	case eventTypeDirection:
		return s.processMoveEvent(msgBytes)
	default:
		return nil, fmt.Errorf("Missing or unknown event type - %s", eventType)
	}
}

func (s *Service) processMoveEvent(msgBytes []byte) (*Actor, error) {
	moveEvent := &DirectionEvent{}
	err := json.Unmarshal(msgBytes, moveEvent)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling direction event - %v", err)
	}
	return nil, nil
}

func (s *Service) RegisterPlayer(ign string) (*Actor, error) {
	startX, startY := s.getPlayerStartPosition()

	player := NewActor(ign, startX, startY)

	return s.addPlayerToMatch(player), nil
}

// returns either the new player or the existing player with this IGN
func (s *Service) addPlayerToMatch(player *Actor) *Actor {
	// TODO: Don't use the concept of current match
	match := s.getCurrentMatch()

	return match.addPlayer(player)
}

// TODO: Don't use the concept of current match
func (s *Service) getCurrentMatch() *Match {
	// Otherwise return the current one
	return s.getMatch(s.CurrentMatchID)
}

// Safelly gets a match from the store
func (s *Service) getMatch(id string) *Match {
	s.mutex.Lock()
	match := s.matches[id]
	s.mutex.Unlock()

	return match
}

func (s *Service) RegisterMatch() *Match {
	match := NewMatch()

	s.matches[match.ID] = match

	// TODO: Don't use the concept of current match
	s.CurrentMatchID = match.ID

	return match
}

// Determines a good starting point for the player based on the current player locations
func (s *Service) getPlayerStartPosition() (int, int) {
	return 0, 0 //TODO
}

func (s *Service) UpdatePlayers(matchID string) map[string]*Actor {
	match := s.getMatch(matchID)

	match.mutex.Lock()
	for _, player := range match.Players {
		// Move the player in their current direction
		s.movePlayer(match, player, player.Direction)
	}
	match.mutex.Unlock()

	return match.Players
}

func (s *Service) movePlayer(match *Match, player *Actor, directionVector []int) {
	if s.canMove(match, player, directionVector) {
		player.move(directionVector)
	}
}

func (s *Service) canMove(match *Match, player *Actor, directionVector []int) bool {
	x, y := s.getPlayerGridPosition(match, player)

	newGridX := x + directionVector[0]
	newGridY := y + directionVector[1]

	tile := match.Map[newGridY][newGridX]
	if tile == 0 {
		return true
	} else if tile == 2 {
		// wrap
	}

	return false
}

// Returns the top left grid position nearest to the player
func (s *Service) getPlayerGridPosition(match *Match, player *Actor) (int, int) {
	return player.X / match.GridSize,
		player.Y / match.GridSize
}
