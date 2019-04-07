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

	log.Info().Msgf("Started match with ID %s", match.ID)

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

// returns the movable distance in each direction
// TODO: just ugh
func (s *Service) canMove(match *Match, player *Actor, directionVector []int) []int {
	gridX, gridY := player.X/gridSize, player.Y/gridSize
	log.Info().Msgf("gridX:%d,gridY:%d", gridX, gridY)

	x2, y2 := player.getPoint2()
	log.Info().Msgf("x2:%d,y2:%d", x2, y2)
	gridX2, gridY2 := x2/gridSize, y2/gridSize
	log.Info().Msgf("gridX2:%d,gridY2:%d", gridX2, gridY2)

	xDist := speed * directionVector[0]
	yDist := speed * directionVector[1]

	if directionVector[0] == -1 { //left
		// check the 2 or three tiles to the left that we could collide with
		for i := gridY; i < gridY2; i++ {
			log.Info().Msgf("moving left - Checking (%d,%d)", i, gridX-1)
			if s.checkTile(match.Map[i][gridX-1]) {
				// returns true if the tile is passable, check the next one
				continue
			}

			// otherwise, we have a potential collision ahead, see how far
			xDist = player.X % gridSize
			log.Info().Msgf("moving left - xDist:%d", xDist)
			if xDist < 0 {
				xDist = 0
			} else if xDist > speed {
				xDist = -speed
			}
		}

	} else if directionVector[0] == 1 { //right
		for i := gridY; i < gridY2; i++ {
			log.Info().Msgf("moving right - Checking (%d,%d)", i, gridX2+1)
			if s.checkTile(match.Map[i][gridX2+1]) {
				continue
			}

			xDist = ((gridX2 + 1) * gridSize) - x2
			log.Info().Msgf("moving right - xDist:%d", xDist)
			if xDist < 0 {
				xDist = 0
			} else if xDist > speed {
				xDist = speed
			}
		}

	} else if directionVector[1] == -1 { //up
		for i := gridX; i < gridX2; i++ {
			log.Info().Msgf("moving up - Checking (%d,%d)", gridY-1, i)
			if s.checkTile(match.Map[gridY-1][i]) {
				continue
			}

			yDist = player.Y % gridSize
			log.Info().Msgf("moving up - yDist:%d", yDist)
			if yDist < 0 {
				yDist = 0
			} else if yDist > speed {
				yDist = -speed
			}
		}

	} else if directionVector[1] == 1 { //down
		for i := gridX; i < gridX2; i++ {
			log.Info().Msgf("moving down - Checking (%d,%d)", gridY2+1, i)
			if s.checkTile(match.Map[gridY2+1][i]) {
				continue
			}

			yDist = ((gridY2 + 1) * gridSize) - y2 - 1
			log.Info().Msgf("moving down - yDist:%d", yDist)
			if yDist < 0 {
				yDist = 0
			} else if yDist > speed {
				yDist = speed
			}
		}
	}

	return []int{xDist, yDist}
}

func (s *Service) checkTile(tile int) bool {
	if tile == 0 { // 0 is floor
		return true
	} else if tile == 2 { // TODO: 2 is screen wrap
		return true
	}

	return false
}
