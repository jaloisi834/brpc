package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

const minPlayers = 2

const keyEventType = "type"

const (
	eventTypeDirection = "direction"
)

type Service struct {
	matches        map[string]*Match
	currentMatchID string // TODO: Don't use the concept of current match
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

func (s *Service) canMove(player *Actor, directionVector []int) bool {
	match := s.getCurrentMatch()

	x, y := s.getPlayerGridPosition(match, player)

	newGridX := x + directionVector[0]
	newGridY := y + directionVector[1]

	if match.Map[newGridY][newGridX] {
		return true
	}

	return false
}

func (s *Service) getPlayerGridPosition(match *Match, player *Actor) (int, int) {
	return player.X / match.GridSize,
		player.Y / match.GridSize
}

func (s *Service) RegisterPlayer(playerID string) (*Actor, error) {
	startX, startY := s.getPlayerStartPosition()

	player := &Actor{
		ID: playerID,
		X:  startX,
		Y:  startY,
	}

	// If there is no match, create a new one
	if s.currentMatchID == "" {
		s.registerMatch()
	}

	s.addPlayerToMatch(player)

	return player, nil
}

func (s *Service) addPlayerToMatch(player *Actor) {
	// TODO: Don't use the concept of current match
	match := s.getCurrentMatch()

	match.Players[player.ID] = player
}

// TODO: Don't use the concept of current match
func (s *Service) getCurrentMatch() *Match {
	// Otherwise return the current one
	return s.matches[s.currentMatchID]
}

func (s *Service) registerMatch() *Match {
	match := NewMatch()

	s.matches[match.ID] = match

	// TODO: Don't use the concept of current match
	s.currentMatchID = match.ID

	return match
}

// Determines a good starting point for the player based on the current player locations
func (s *Service) getPlayerStartPosition() (int, int) {
	return 0, 0 //TODO
}

func (s *Service) updatePlayerPosition(matchID, playerID string, x, y int) error {
	match, ok := s.matches[matchID]
	if !ok {
		return errors.New("match not found")
	}

	player, ok := match.Players[playerID]
	if !ok {
		return errors.New("player not found in match")
	}

	player.setPosition(x, y)

	return nil
}
