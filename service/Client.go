package service

import (
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
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
		h.Broadcast <- BroadcastMessage{msg, c}
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
