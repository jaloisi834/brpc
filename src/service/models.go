package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
)

// TODO: Make dynamic?
const gridSize = 23

const speed = 10

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

type Actor struct {
	ID        string `json:"id"`
	IGN       string `json:"ign"`
	Eating    int    `json:"eating"`
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

func (a *Actor) getPoint2() (int, int) {
	return a.X + (gridSize * 2),
		a.Y + (gridSize * 2)
}

type Match struct {
	mutex    sync.Mutex // Protect players access
	ID       string
	GridSize int
	Map      [][]int
	Players  map[string]*Actor // [playerID]Actor
}

func NewMatch() *Match {
	//TODO: build map
	return &Match{
		GridSize: gridSize,
		Map:      loadMap(),
		ID:       uuid.Must(uuid.NewV4()).String(),
		Players:  make(map[string]*Actor),
	}
}

// TODO: Don't hardcode map location and handle errors without dying
func loadMap() [][]int {
	mapFile, err := os.Open("./maps/map1.pacm")
	if err != nil {
		log.Fatal().Msg("Error loading map")
	}
	defer mapFile.Close()
	mapReader := bufio.NewReader(mapFile)

	mapSizeBytes, err := mapReader.ReadBytes('\n')
	if err != nil {
		log.Fatal().Msg("Error reading size line from map file")
	}

	mapSizeParts := strings.Split(string(mapSizeBytes), ":")
	if len(mapSizeParts) < 2 {
		log.Fatal().Msg("Invalid map size format")
	}

	mapSizeY, err := strconv.Atoi(mapSizeParts[0])
	mapSizeX, err := strconv.Atoi(mapSizeParts[1][:2])
	if err != nil {
		log.Fatal().Str("mapSizeString", string(mapSizeBytes)).Msg("Error converting map size parts to ints")
	}

	log.Debug().Msgf("Map size - %d:%d", mapSizeY, mapSizeX)

	m := make([][]int, mapSizeY)

	// Iterate through all of the characters and set 1's to true
	for y := 0; y < mapSizeY; y++ {
		m[y] = make([]int, mapSizeX)

		for x := 0; x < mapSizeX; x++ {
			r, err := mapReader.ReadByte()
			if err != nil {
				if err == io.EOF {
					return m
				} else {
					log.Fatal().Err(err).Msg("Error reading from map")
				}
			}

			if r == '1' {
				m[y][x] = 1
			} else if r == '2' {
				m[y][x] = 2
			}
		}
	}

	fmt.Printf("%v", m)

	log.Debug().Msg("Successfully loaded map")
	return m
}

// Safely gets a player from the players map
func (m *Match) getPlayer(playerID string) *Actor {
	m.mutex.Lock()
	player := m.Players[playerID]
	m.mutex.Unlock()

	return player
}

func (m *Match) addPlayer(player *Actor) *Actor {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, existingPlayer := range m.Players {
		if existingPlayer.IGN == player.IGN {
			return existingPlayer
		}
	}

	m.Players[player.ID] = player
	return player
}

type Payload struct {
	Type    string      `json:"eventType"`
	Tick    int64       `json:"tick"`
	MatchID string      `json:"matchId"`
	Data    interface{} `json:"data"`
}
