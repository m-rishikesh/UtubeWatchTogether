package handlers

import (
	"gotth/service"
	"log"
	"net/http"

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

	var targetHub *service.Hub

	// If room code is provided, find the room; otherwise use default hub
	if roomCode != "" {
		room := hm.FindRoom(roomCode)
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

	client := &service.Client{Conn: conn, Send: make(chan []byte, 256)}
	targetHub.Register <- client
	log.Println("Connected:", client, "to room:", roomCode)

	go client.SendToHub(targetHub)
	go client.ReceiveFromHub(targetHub)

	return nil
}
