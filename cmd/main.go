package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jaloisi834/brpc/src/service"
	"github.com/rs/zerolog/log"
)

const registrationPath = "/register"

const serverPort = 8080

const tickRate = 2000 // milliseconds

var connections = make(map[string]*websocket.Conn) //[ign]websocketConnection
var mutex sync.Mutex                               // Protects connection writes

func main() {
	s := service.New()

	http.HandleFunc(registrationPath, func(w http.ResponseWriter, r *http.Request) {
		ign := r.URL.Query().Get("ign")

		// Setup a websocket connection for the player
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			log.Error().Err(err).Msg("Error creating websocket connection for player")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error creating websocket connection"))
			return
		}
		connections[ign] = conn

		// Register and return the player to the client
		player, err := s.RegisterPlayer(ign)
		if err != nil {
			log.Error().Err(err).Msg("Error registering player")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error registering player"))
			return
		}

		log.Info().Interface("actor", player).Msgf("Registered player - %s", ign)

		payload := service.Payload{
			Type:    "registered",
			MatchID: s.CurrentMatchID,
			Data:    player,
		}
		err = writeJSONToConn(conn, payload)
		if err != nil {
			log.Error().Err(err).Msg("Error sending player JSON over websocket")
		}

		// Start an indefinite loop to handle events on this connection
		handleEvents(ign, conn, s)
		log.Info().Msgf("Closing connection for player %s", ign)
	})

	// Create a new match
	s.RegisterMatch()

	// Start up the ticker to update players in the background
	var ticker = time.NewTicker(tickRate * time.Millisecond)
	go tick(ticker, s)
	log.Info().Msg("Started ticker")

	serverAddress := fmt.Sprintf(":%d", serverPort)
	log.Info().Msgf("Starting server at %s", serverAddress)
	http.ListenAndServe(serverAddress, nil)
}

func handleEvents(ign string, conn *websocket.Conn, service *service.Service) {
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Error reading message from client: Closing connection")
			closeConnection(ign, conn)
			return
		}

		if msgType == websocket.CloseMessage {
			log.Info().Msg("Received close signal from client: Closing connection")
			closeConnection(ign, conn)
			return
		}

		log.Info().
			Interface("remoteAddress", conn.RemoteAddr()).
			Str("msg", string(msg)).
			Msg("Received message from client")

		// process the event
		err = service.ProcessEvent(msg)
		if err != nil {
			log.Error().Err(err).Msg("Error processing event")
		}
	}
}

func closeConnection(ign string, conn *websocket.Conn) {
	delete(connections, ign)

	err := conn.Close()
	if err != nil {
		log.Error().Err(err).Msgf("Error closing connection for player - %s", ign)
	}
}

func tick(ticker *time.Ticker, s *service.Service) {
	for t := range ticker.C {
		players := s.UpdatePlayers(s.CurrentMatchID)
		payload := service.Payload{
			Type:    "frame",
			Tick:    t.UnixNano(),
			MatchID: s.CurrentMatchID,
			Data:    players,
		}
		broadcastEvent(payload)
	}
}

// Send an event to all players
func broadcastEvent(event interface{}) {
	for ign, conn := range connections {
		// concurrently write to the connection
		go func(ign string, conn *websocket.Conn) {
			err := writeJSONToConn(conn, event)
			if err != nil {
				log.Error().Err(err).Msgf("Error sending event to player - %s", ign)
				closeConnection(ign, conn)
			}
		}(ign, conn)
	}
}

// Safely writes JSON to the connection
func writeJSONToConn(conn *websocket.Conn, event interface{}) error {
	mutex.Lock()
	err := conn.WriteJSON(event)
	mutex.Unlock()

	return err
}
