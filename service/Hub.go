package service

import (
	"encoding/json"
	"fmt"
)

type Hub struct {
	Clients       map[*Client]bool
	Register      chan *Client
	Unregister    chan *Client
	Broadcast     chan BroadcastMessage
	PlayerTime    float64
	LastUpdatedAt int64
	IsPlaying     bool
	StateMutex    chan bool // Used to safely update state
}

type VideoState struct {
	Type          string  `json:"type"`
	CurrentTime   float64 `json:"currentTime,omitempty"`
	IsPlaying     bool    `json:"isPlaying,omitempty"`
	LastUpdatedAt int64   `json:"sendAt,omitempty"`
}

type BroadcastMessage struct {
	Message []byte
	Sender  *Client
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan BroadcastMessage),
		StateMutex: make(chan bool, 1),
	}
}

func (h *Hub) Run() {
	h.StateMutex <- true // Initialize semaphore
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			// Send current state to newly connected client
			<-h.StateMutex
			state := VideoState{
				Type:          "sync",
				CurrentTime:   h.PlayerTime,
				LastUpdatedAt: h.LastUpdatedAt, // Send current time, not old timestamp
				IsPlaying:     h.IsPlaying,
			}
			h.StateMutex <- true
			fmt.Println("client joined:", state.Type, state.CurrentTime, state.LastUpdatedAt, state.IsPlaying)
			if data, err := json.Marshal(state); err == nil {
				select {
				case client.Send <- data:
				default:
				}
			}

		case client := <-h.Unregister:
			delete(h.Clients, client)
			close(client.Send)

		case message := <-h.Broadcast:
			// Parse incoming message to update state
			var msg VideoState
			messageData := message.Message
			if err := json.Unmarshal(messageData, &msg); err == nil {
				<-h.StateMutex
				if msg.Type == "play" {
					h.IsPlaying = true
					h.PlayerTime = msg.CurrentTime
					h.LastUpdatedAt = msg.LastUpdatedAt
				} else if msg.Type == "pause" {
					h.IsPlaying = false
					h.PlayerTime = msg.CurrentTime
					h.LastUpdatedAt = msg.LastUpdatedAt
				} else if msg.Type == "seek" {
					h.PlayerTime = msg.CurrentTime
					h.LastUpdatedAt = msg.LastUpdatedAt
				}
				h.StateMutex <- true
			}

			// Broadcast to all clients EXCEPT sender
			for client := range h.Clients {
				// Skip the sender
				if client == message.Sender {
					fmt.Println("Skipping sender client")
					continue
				}

				select {
				case client.Send <- messageData:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}
