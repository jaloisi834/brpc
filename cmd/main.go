package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jaloisi834/brpc/src/service"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
)

const registrationPath = "/register"

const serverPort = 8080

var connections = make(map[string]*websocket.Conn) //[playerID]websocketConnection

func main() {
	service := service.New()

	http.HandleFunc(registrationPath, func(w http.ResponseWriter, r *http.Request) {
		playerID := uuid.Must(uuid.NewV4()).String()

		// Setup a websocket connection for the player
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			log.Error().Err(err).Msg("Error creating websocket connection for player")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error creating websocket connection"))
			return
		}
		connections[playerID] = conn

		// Register and return the player to the client
		player, err := service.RegisterPlayer(playerID)
		if err != nil {
			log.Error().Err(err).Msg("Error registering player")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error registering player"))
			return
		}

		err = conn.WriteJSON(player)
		if err != nil {
			log.Error().Err(err).Msg("Error sending player JSON over websocket")
		}

		// Start an indefinite loop to handle events on this connection
		handleEvents(playerID, conn, service)
		log.Info().Msgf("Closing connection for player %s", playerID)
	})

	serverAddress := fmt.Sprintf(":%d", serverPort)
	log.Info().Msgf("Starting server at %s", serverAddress)
	http.ListenAndServe(serverAddress, nil)
}

func handleEvents(playerID string, conn *websocket.Conn, service *service.Service) {
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Error reading message from client")
		}

		if msgType == websocket.CloseMessage {
			return
		}

		log.Info().
			Interface("remoteAddress", conn.RemoteAddr()).
			Str("msg", string(msg)).
			Msg("Received message from client")

		// process the event
		service.ProcessEvent(msg)

		// Send it to everyone
		broadcastEvent(msg)
	}
}

// Send an event to all players
func broadcastEvent(event interface{}) {
	for playerID, conn := range connections {

		// concurrently write to the connection
		go func(playerID string, conn *websocket.Conn) {
			err := conn.WriteJSON(event)
			if err != nil {
				log.Error().Err(err).Msgf("Error sending event to player - %s", playerID)
			}
		}(playerID, conn)
	}
}
