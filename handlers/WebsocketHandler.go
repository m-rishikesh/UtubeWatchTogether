package handlers

import (
	"encoding/json"
	"fmt"
	"gotth/service"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
)

var Hub = service.NewHub()

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func WebsktHandler(c *echo.Context, hm *service.HubManager) error {
	// Get room code from query parameter
	roomCode := c.QueryParam("code")
	fmt.Println("roomcode:", roomCode)
	var targetHub *service.Hub

	// If room code is provided, find the room; otherwise use default hub
	room := hm.FindRoom(roomCode)
	if roomCode != "" {
		if room == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
		}
		targetHub = room.Hub
	} else {
		targetHub = Hub
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return err
	}

	client := &service.Client{Conn: conn, Send: make(chan []byte, 256), Room: room}
	targetHub.Register <- client
	log.Println("[CONNECTED]:", client, "to room:", roomCode)

	go client.SendToHub(targetHub)
	go client.ReceiveFromHub(targetHub)

	return nil
}

// contains webRTC logic

type SignalMessage struct {
	Type      string          `json:"type"`
	From      string          `json:"from,omitempty"`
	To        string          `json:"to,omitempty"`
	UserID    string          `json:"userId,omitempty"`
	Users     []string        `json:"users,omitempty"`
	Offer     json.RawMessage `json:"offer,omitempty"`
	Answer    json.RawMessage `json:"answer,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
}

func generateID() string {
	return strconv.Itoa(rand.Intn(1000000))
}

func WsEchoHandler(c *echo.Context, hm *service.HubManager) error {
	wsHandler(c.Response(), c.Request(), hm)
	return nil
}

func wsHandler(w http.ResponseWriter, r *http.Request, hm *service.HubManager) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	roomcode := r.FormValue("code")
	id := generateID()
	room := hm.FindRoom(roomcode)
	fmt.Println("Room:", room)
	if room == nil {
		conn.Close()
		return
	}
	client := &service.RTCClient{
		ID:   id,
		Conn: conn,
		Mu:   &sync.Mutex{},
		Room: room,
	}
	fmt.Println("CLient Video:", client.ID)
	room.Mu.Lock()

	existingUsers := []string{}
	fmt.Println("Room Clients:", room.Clients)
	for userID := range room.Clients {
		existingUsers = append(existingUsers, userID)
	}

	room.Clients[id] = client

	room.Mu.Unlock()

	client.WriteJSON(SignalMessage{
		Type:   "userId",
		UserID: client.ID,
	})

	client.WriteJSON(SignalMessage{
		Type:  "existing_user",
		Users: existingUsers,
	})

	broadcastUserJoined(client, room)

	defer func() {

		room.Mu.Lock()
		delete(room.Clients, id)
		room.Mu.Unlock()

		broadcastUserLeft(client, room)

		conn.Close()
	}()

	for {

		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg SignalMessage

		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {

		case "offer", "answer", "candidate":

			room.Mu.RLock()
			target, ok := room.Clients[msg.To]
			room.Mu.RUnlock()

			if ok {
				target.WriteMessage(
					websocket.TextMessage,
					data,
				)
			}
		}
	}
}

func broadcastUserJoined(client *service.RTCClient, room *service.Room) {

	msg := SignalMessage{
		Type:   "user_joined",
		UserID: client.ID,
	}

	room.Mu.RLock()

	targets := make([]*service.RTCClient, 0, len(room.Clients))

	for userID, c := range room.Clients {
		if userID == client.ID {
			continue
		}
		targets = append(targets, c)
	}

	room.Mu.RUnlock()

	for _, c := range targets {
		c.WriteJSON(msg)
	}
}

func broadcastUserLeft(client *service.RTCClient, room *service.Room) {

	msg := SignalMessage{
		Type:   "user_left",
		UserID: client.ID,
	}

	room.Mu.RLock()

	targets := make([]*service.RTCClient, 0, len(room.Clients))

	for _, c := range room.Clients {
		targets = append(targets, c)
	}

	room.Mu.RUnlock()

	for _, c := range targets {
		c.WriteJSON(msg)
	}
}
