package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jaloisi834/brpc/src/service"
	"github.com/rs/zerolog/log"
)

const registrationPath = "/register"

const tickRate = 1000 // milliseconds

type Handler struct {
	connections    map[string]*websocket.Conn //[ign]websocketConnection
	service        *service.Service
	mutex          sync.Mutex // Protects connection writes
	currentMatchID string     // TODO: Don't use the concept of current match
}

func New(service *service.Service, currentMatchID string) *Handler {
	return &Handler{
		service:        service,
		connections:    make(map[string]*websocket.Conn),
		currentMatchID: currentMatchID,
	}
}

func (h *Handler) Registration(w http.ResponseWriter, r *http.Request) {
	ign := r.URL.Query().Get("ign")

	// Setup a websocket connection for the player
	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		log.Error().Err(err).Msg("Error creating websocket connection for player")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error creating websocket connection"))
		return
	}
	h.connections[ign] = conn

	// Register and return the player to the client
	player, err := h.service.RegisterPlayer(h.currentMatchID, ign)
	if err != nil {
		log.Error().Err(err).Msg("Error registering player")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error registering player"))
		return
	}

	log.Info().Interface("actor", player).Msgf("Registered player - %s", ign)

	payload := service.Payload{
		Type:    "registered",
		MatchID: h.currentMatchID,
		Data:    player,
	}
	err = h.writeJSONToConn(conn, payload)
	if err != nil {
		log.Error().Err(err).Msg("Error sending player JSON over websocket")
	}

	// Start an indefinite loop to handle events on this connection
	h.handleEvents(ign, conn)
	log.Info().Msgf("Closing connection for player %s", ign)
}

func (h *Handler) handleEvents(ign string, conn *websocket.Conn) {
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Error reading message from client: Closing connection")
			h.closeConnection(ign, conn)
			return
		}

		if msgType == websocket.CloseMessage {
			log.Info().Msg("Received close signal from client: Closing connection")
			h.closeConnection(ign, conn)
			return
		}

		log.Info().
			Interface("remoteAddress", conn.RemoteAddr()).
			Str("msg", string(msg)).
			Msg("Received message from client")

		// process the event
		err = h.service.ProcessEvent(msg)
		if err != nil {
			log.Error().Err(err).Msg("Error processing event")
		}
	}
}

func (h *Handler) closeConnection(ign string, conn *websocket.Conn) {
	delete(h.connections, ign)

	err := conn.Close()
	if err != nil {
		log.Error().Err(err).Msgf("Error closing connection for player - %s", ign)
	}
}

func (h *Handler) StartTicker() {
	var ticker = time.NewTicker(tickRate * time.Millisecond)
	go h.tick(ticker)
	log.Info().Msg("Started ticker")
}

func (h *Handler) tick(ticker *time.Ticker) {
	for t := range ticker.C {
		players := h.service.UpdatePlayers(h.currentMatchID)

		payload := service.Payload{
			Type:    "frame",
			Tick:    t.UnixNano(),
			MatchID: h.currentMatchID,
			Data:    players,
		}

		h.broadcastEvent(payload)
	}
}

// Send an event to all players
func (h *Handler) broadcastEvent(event interface{}) {
	for ign, conn := range h.connections {
		// concurrently write to the connection
		go func(ign string, conn *websocket.Conn) {
			err := h.writeJSONToConn(conn, event)
			if err != nil {
				log.Error().Err(err).Msgf("Error sending event to player - %s", ign)
				h.closeConnection(ign, conn)
			}
		}(ign, conn)
	}
}

// Safely writes JSON to the connection
func (h *Handler) writeJSONToConn(conn *websocket.Conn, event interface{}) error {
	h.mutex.Lock()
	err := conn.WriteJSON(event)
	h.mutex.Unlock()

	return err
}
