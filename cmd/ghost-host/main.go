package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jaloisi834/ghost-host/internal/handler"
	"github.com/jaloisi834/ghost-host/internal/service"
	"github.com/rs/zerolog/log"
)

const registrationPath = "/register"

const serverPort = 8080

var connections = make(map[string]*websocket.Conn) //[ign]websocketConnection
var mutex sync.Mutex                               // Protects connection writes

var currentMatchID = "" // TODO: Don't use the concept of current match

func main() {
	s := service.New()

	// Create a new match
	m := s.RegisterMatch()

	h := handler.New(s, m.ID)
	http.HandleFunc(registrationPath, h.Registration)

	// Start up the ticker to update players in the background
	h.StartTicker()

	serverAddress := fmt.Sprintf(":%d", serverPort)
	log.Info().Msgf("Starting server at %s", serverAddress)
	http.ListenAndServe(serverAddress, nil)
}
