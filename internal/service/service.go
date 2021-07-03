package service

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

// TODO: Use?
const minPlayers = 2

const keyEventType = "eventType"

const (
	eventTypeDirection = "direction"
)

type Service struct {
	mutex   sync.Mutex // Protect matches access
	matches map[string]*Match
}

func New() *Service {
	return &Service{
		matches: make(map[string]*Match, 1),
	}
}

func (s *Service) ProcessEvent(msgBytes []byte) error {
	msg := string(msgBytes)
	log.Info().Str("message", msg).Msg("Received client message")

	// Take action based on the event type
	eventType := gjson.Get(msg, keyEventType).String()
	switch eventType {
	case eventTypeDirection:
		return s.processDirectionEvent(msgBytes)
	default:
		return fmt.Errorf("Missing or unknown event type - %s", eventType)
	}
}

func (s *Service) processDirectionEvent(msgBytes []byte) error {
	directionEvent := &DirectionEvent{}
	err := json.Unmarshal(msgBytes, directionEvent)
	if err != nil {
		return fmt.Errorf("Error unmarshalling direction event - %v", err)
	}

	match := s.getMatch(directionEvent.MatchID)
	if match == nil {
		return fmt.Errorf("Match not found - %s", directionEvent.MatchID)
	}

	player := match.getPlayer(directionEvent.PlayerID)
	if player == nil {
		return fmt.Errorf("Player not found - %s", directionEvent.PlayerID)
	}

	s.turnPlayer(match, player, directionEvent.NewDirection)

	return nil
}

// RegisterPlayer creates a new player with the given IGN and attempts to assign it to a match
// If a player already exists with the given IGN, the existing record will be returned instead
func (s *Service) RegisterPlayer(matchID, ign string) (*Actor, error) {
	match := s.getMatch(matchID)
	if match == nil {
		return nil, fmt.Errorf("Match not found - %s", matchID)
	}

	startX, startY := s.getPlayerStartPosition(match)

	player := NewActor(ign, startX, startY)

	return s.addPlayerToMatch(match, player)
}

// returns either the new player or the existing player with this IGN
func (s *Service) addPlayerToMatch(match *Match, player *Actor) (*Actor, error) {
	return match.addPlayer(player), nil
}

// Safelly gets a match from the store
func (s *Service) getMatch(id string) *Match {
	s.mutex.Lock()
	match := s.matches[id]
	s.mutex.Unlock()

	return match
}

// RegisterMatch creates a new match
func (s *Service) RegisterMatch() *Match {
	match := NewMatch()

	s.matches[match.ID] = match

	log.Info().Msgf("Started match with ID %s", match.ID)

	return match
}

// Determines a good starting point for the player based on the current player locations
func (s *Service) getPlayerStartPosition(match *Match) (float32, float32) {
	startPosition := match.getNextStartPosition()

	return startPosition.X * gridSize,
		startPosition.Y * gridSize
}

func (s *Service) UpdatePlayers(matchID string) map[string]*Actor {
	match := s.getMatch(matchID)

	match.mutex.Lock()
	for _, player := range match.Players {
		// Don't check dead players
		if player.Dead {
			continue
		}

		// Move the player in their current direction
		s.movePlayer(match, player, player.Direction)
	}
	match.mutex.Unlock()

	return match.Players
}

func (s *Service) movePlayer(match *Match, player *Actor, directionVector []int) {
	distance := s.canMove(match, player, directionVector)

	if directionVector[0] != 0 && distance[0] == 0 {
		player.Direction[0] = 0
	}

	if directionVector[1] != 0 && distance[1] == 0 {
		player.Direction[1] = 0
	}

	player.move(distance)
}

func (s *Service) turnPlayer(match *Match, player *Actor, directionVector []int) {
	distance := s.canMove(match, player, directionVector)

	// If the direction we want to go and the associated canMove distance are non-zero
	// it means that we can move in that direction. Update the player's direction for the next tick to process
	if (directionVector[0] != 0 && distance[0] != 0) ||
		(directionVector[1] != 0 && distance[1] != 0) {
		log.Debug().Interface("actor", player).Msg("Player turned")
		player.Direction = directionVector
	}
}

func (s *Service) checkCollision(match *Match, player *Actor) {
	match.mutex.Lock()
	for _, otherPlayer := range match.Players {
		if s.intersect(player, otherPlayer) {
			// Don't check dead players
			if otherPlayer.Dead {
				continue
			}

			playerEating := player.Eating
			otherPlayerEating := otherPlayer.Eating

			// If they are equal in power, bounce them off each other
			if playerEating == otherPlayerEating {
				player.reverseDirection()
				otherPlayer.reverseDirection()
				continue
			}

			// Otherwise process tha damage done
			player.processDamage(otherPlayerEating)
			otherPlayer.processDamage(playerEating)
		}
	}
	match.mutex.Unlock()
}

func (s *Service) intersect(a1, a2 *Actor) bool {
	a1X, a1Y := a1.X, a1.Y
	a1X2, a1Y2 := a1.getPoint2()

	a2X, a2Y := a2.X, a2.Y
	a2X2, a2Y2 := a2.getPoint2()

	if a1X > a2X2 ||
		a1X2 < a2X ||
		a1Y < a2Y2 ||
		a1Y2 > a2Y {
		return false
	}

	return true
}

// returns the movable distance in each direction
// TODO: just ugh
func (s *Service) canMove(match *Match, player *Actor, directionVector []int) []float32 {
	gridX, gridY := int(player.X/gridSize), int(player.Y/gridSize)

	x2, y2 := player.getPoint2()
	gridX2, gridY2 := int(x2/gridSize), int(y2/gridSize)

	xDist := float32(speed * directionVector[0])
	yDist := float32(speed * directionVector[1])

	if directionVector[0] == -1 { //left
		// check the 2 or three tiles to the left that we could collide with
		for i := gridY; i < gridY2; i++ {
			if gridX-1 <= 0 {
				xDist = 0
				continue
			}
			if s.checkTile(match.Map[i][gridX-1]) {
				// returns true if the tile is passable, check the next one
				continue
			}

			// otherwise, we have a potential collision ahead, see how far
			xDist = player.X - (float32(gridX) * gridSize)
			if xDist < 0 {
				xDist = 0
			} else if xDist > speed {
				xDist = -speed
			}
		}

	} else if directionVector[0] == 1 { //right
		for i := gridY; i < gridY2; i++ {
			log.Info().Msgf("moving right - Checking (%d,%d)", i, gridX2+1)
			if gridX2+1 >= len(match.Map[i]) {
				xDist = 0
				continue
			}
			if s.checkTile(match.Map[i][gridX2+1]) {
				continue
			}

			xDist = (float32(gridX2+1) * gridSize) - x2
			if xDist < 0 {
				xDist = 0
			} else if xDist > speed {
				xDist = speed
			}
		}

	} else if directionVector[1] == -1 { //up
		for i := gridX; i < gridX2; i++ {
			log.Info().Msgf("moving up - Checking (%d,%d)", gridY-1, i)
			if gridY-1 <= 0 {
				yDist = 0
				continue
			}
			if s.checkTile(match.Map[gridY-1][i]) {
				continue
			}

			yDist = player.Y - (float32(gridY) * gridSize)
			if yDist < 0 {
				yDist = 0
			} else if yDist > speed {
				yDist = -speed
			}
		}

	} else if directionVector[1] == 1 { //down
		for i := gridX; i < gridX2; i++ {
			log.Info().Msgf("moving down - Checking (%d,%d)", gridY2+1, i)
			if gridY2+1 >= len(match.Map) {
				yDist = 0
				continue
			}
			if s.checkTile(match.Map[gridY2+1][i]) {
				continue
			}

			yDist = (float32(gridY2+1) * gridSize) - y2 - 1
			if yDist < 0 {
				yDist = 0
			} else if yDist > speed {
				yDist = speed
			}
		}
	}

	return []float32{xDist, yDist}
}

func (s *Service) checkTile(tile int) bool {
	if tile == 0 { // 0 is floor
		return true
	} else if tile == 2 { // TODO: 2 is screen wrap
		return true
	}

	return false
}
