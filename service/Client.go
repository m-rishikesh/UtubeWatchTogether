package service

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
	Room *Room
	Once sync.Once
}

type ChatMessage struct {
	User    string `json:"user"`
	Message string `json:"message"`
}

type VideoMessage struct {
	Type        string  `json:"type"`
	CurrentTime float64 `json:"currentTime"`
	SendAt      int64   `json:"sendAt"`
	IsPlaying   bool    `json:"isPlaying,omitempty"`
}

type WSMessage struct {
	VideoMessage *VideoMessage `json:"videomessage"`
	ChatMessage  *ChatMessage  `json:"chatmessage"`
}

func (c *Client) Disconnect() {
	if c.Room == nil {
		log.Println("from disconnect: Room is nil")
		return
	}

	if c.Room.Hub == nil {
		log.Println("Hub is nil")
		return
	}
	c.Once.Do(func() {
		select {
		case c.Room.Hub.Unregister <- c:
			fmt.Printf("[DISCONNECTED]: client: %v disconnected from room: %v", c, c.Room)
		default:
		}
		c.Conn.Close()
	})
}

func (c *Client) SendToHub(h *Hub) {
	defer c.Disconnect()
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		log.Println("Received:", string(msg))
		var wsmsg WSMessage
		json.Unmarshal(msg, &wsmsg)

		switch {
		case wsmsg.ChatMessage != nil:
			h.Broadcast <- BroadcastMessage{
				MessageChat: msg,
				Sender:      c,
			}
		case wsmsg.VideoMessage != nil:
			h.Broadcast <- BroadcastMessage{
				MessageVideo: msg,
				Sender:       c,
			}
		}
	}

}

func (c *Client) ReceiveFromHub(h *Hub) {
	defer c.Disconnect()
	for {
		message, ok := <-c.Send
		if !ok {
			return
		}

		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			c.Disconnect()
			return
		}
	}
}
