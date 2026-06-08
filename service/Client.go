package service

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
	Room *Room
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

func (c *Client) SendToHub(h *Hub) {
	defer func() {
		h.Unregister <- c
		c.Conn.Close()
	}()

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
	defer c.Conn.Close()
	for {
		message, ok := <-c.Send
		if !ok {
			return
		}

		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
}
