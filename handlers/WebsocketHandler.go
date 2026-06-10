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

type Client struct {
	ID   string
	Conn *websocket.Conn
	Mu   *sync.Mutex
}

func (c *Client) WriteJSON(v any) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	return c.Conn.WriteJSON(v)
}

func (c *Client) WriteMessage(mt int, data []byte) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	return c.Conn.WriteMessage(mt, data)
}

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

var (
	clients = make(map[string]*Client)
	mu      sync.RWMutex
)

func generateID() string {
	return strconv.Itoa(rand.Intn(1000000))
}

func WsEchoHandler(c *echo.Context) error {
	wsHandler(c.Response(), c.Request())
	return nil
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	id := generateID()

	client := &Client{
		ID:   id,
		Conn: conn,
		Mu:   &sync.Mutex{},
	}

	mu.Lock()

	existingUsers := []string{}
	for userID := range clients {
		existingUsers = append(existingUsers, userID)
	}

	clients[id] = client

	mu.Unlock()

	conn.WriteJSON(SignalMessage{
		Type:   "userId",
		UserID: id,
	})

	broadcastUserJoined(id)

	defer func() {

		mu.Lock()
		delete(clients, id)
		mu.Unlock()

		broadcastUserLeft(id)

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

			mu.RLock()
			target, ok := clients[msg.To]
			mu.RUnlock()

			if ok {
				target.Conn.WriteMessage(
					websocket.TextMessage,
					data,
				)
			}
		}
	}
}

func broadcastUserJoined(id string) {

	msg := SignalMessage{
		Type:   "user_joined",
		UserID: id,
	}

	mu.RLock()
	defer mu.RUnlock()

	for userID, client := range clients {

		if userID == id {
			continue
		}

		client.Conn.WriteJSON(msg)
	}
}

func broadcastUserLeft(id string) {

	msg := SignalMessage{
		Type:   "user_left",
		UserID: id,
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, client := range clients {
		client.Conn.WriteJSON(msg)
	}
}
