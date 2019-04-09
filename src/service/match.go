package service

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
)

// TODO: Make dynamic?
const gridSize = 26.2 //26.2

// TODO: It's currently assumed that main will be run from the base directory
const map1Path = "./maps/map1.pacm"

const (
	wall = 1
	wrap = 2
)

type Match struct {
	mutex                   sync.Mutex // Protect players access
	ID                      string
	GridSize                float32
	Map                     [][]int           // [y][x]
	Players                 map[string]*Actor // [playerID]Actor
	AvailableStartPositions []startPosition
}

func NewMatch() *Match {
	return &Match{
		GridSize: gridSize,
		Map:      loadMap(),
		ID:       uuid.Must(uuid.NewV4()).String(),
		Players:  make(map[string]*Actor),
		AvailableStartPositions: []startPosition{
			{8, 10},
			{16, 10},
			{8, 17},
			{17, 17},
		},
	}
}

func (m *Match) getNextStartPosition() startPosition {
	if len(m.AvailableStartPositions) == 0 {
		log.Error().Msg("No availible start positions")
		return startPosition{0, 0}
	}

	// Always try to get the first one
	startPosition := m.AvailableStartPositions[0]

	// Then remove it from the slice
	m.AvailableStartPositions = m.AvailableStartPositions[1:]

	return startPosition
}

// TODO: Don't hardcode map location and handle errors without dying
func loadMap() [][]int {
	mapFile, err := os.Open(map1Path)
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

	m := make([][]int, mapSizeY)

	// Iterate through all of the characters and set 1's to true
	for y := 0; y < mapSizeY; y++ {
		m[y] = make([]int, mapSizeX)

		for x := 0; x < mapSizeX; x++ {
			r, err := mapReader.ReadByte()
			if err != nil {
				// TODO: We probably should handle an EOF here
				log.Fatal().Err(err).Msg("Error reading from map")
			}

			// If we get a new line, keep the x index the same
			if r == '\n' {
				x--
			}

			if r == '1' {
				m[y][x] = wall
			} else if r == '2' {
				m[y][x] = wrap
			}
		}
	}

	log.Info().Msg("Successfully loaded map")
	return m
}

// Safely gets a player from the players map
func (m *Match) getPlayer(playerID string) *Actor {
	m.mutex.Lock()
	player := m.Players[playerID]
	m.mutex.Unlock()

	return player
}

// adds a player if a player with the same name doesn't already exist
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

type startPosition struct {
	X float32
	Y float32
}
