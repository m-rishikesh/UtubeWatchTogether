package service

import (
	"encoding/json"
	"fmt"
	"gotth/config"
	"log"
	"sync"
	"time"
)

type Hub struct {
	Clients       map[*Client]bool
	Register      chan *Client
	Unregister    chan *Client
	Broadcast     chan BroadcastMessage
	PlayerTime    float64
	LastUpdatedAt int64
	IsPlaying     bool
	VideoURL      string
	StateMutex    chan bool // Used to safely update state
	Done          chan struct{}
	DoneOnce      sync.Once
}

type VideoState struct {
	Type          string  `json:"type"`
	CurrentTime   float64 `json:"currentTime"`
	IsPlaying     bool    `json:"isPlaying"`
	LastUpdatedAt int64   `json:"sendAt"`
	VideoURL      string  `json:"video_url"`
}

type BroadcastMessage struct {
	MessageVideo []byte
	MessageChat  []byte
	HeartBeat    []byte
	Sender       *Client
}

type Syncwrapper struct {
	MessageVideo *VideoState `json:"videomessage"`
	ChatMessage  any         `json:"chatmessage"`
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan BroadcastMessage),
		StateMutex: make(chan bool, 1),
		Done:       make(chan struct{}),
	}
}

func (h *Hub) ClientSize() int {
	return len(h.Clients)
}

func (h *Hub) HeartBeat() {
	<-h.StateMutex
	state := &VideoState{
		Type:          "heartbeat",
		IsPlaying:     h.IsPlaying,
		LastUpdatedAt: h.LastUpdatedAt,
		CurrentTime:   h.PlayerTime,
		VideoURL:      h.VideoURL,
	}
	h.StateMutex <- true
	beatbyte, err := json.Marshal(state)
	if err != nil {
		log.Println("failed to marshal heartbeat:", err)
		return
	}
	beatMessage := BroadcastMessage{
		MessageVideo: nil,
		MessageChat:  nil,
		HeartBeat:    beatbyte,
		Sender:       nil,
	}

	h.Broadcast <- beatMessage
}

func (h *Hub) HeartBeatLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.HeartBeat()

		case <-h.Done:
			return
		}
	}
}

func (h *Hub) Run() {
	h.StateMutex <- true // Initialize semaphore
	go h.HeartBeatLoop()
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			// Send current state to newly connected client
			config.PersistRoom(client.Room.Code)
			<-h.StateMutex

			state := VideoState{
				Type:          "sync",
				CurrentTime:   h.PlayerTime,
				LastUpdatedAt: h.LastUpdatedAt, // Send current time, not old timestamp
				IsPlaying:     h.IsPlaying,
				VideoURL:      h.VideoURL,
			}
			h.StateMutex <- true
			fmt.Println("client joined:", state.Type, state.CurrentTime, state.LastUpdatedAt, state.IsPlaying)
			syncMessage := Syncwrapper{
				MessageVideo: &state,
				ChatMessage:  nil,
			}
			if data, err := json.Marshal(syncMessage); err == nil {
				select {
				case client.Send <- data:
				default:
				}
			}

		case client := <-h.Unregister:
			delete(h.Clients, client)
			close(client.Send)

			if h.ClientSize() == 0 {

				room := client.Room
				config.SetRoomTTL(room.Code)
				go func() {
					time.Sleep(30 * time.Second)
					if room.Hub.ClientSize() != 0 {
						return
					}
					room.Manager.Mutex.Lock()
					delete(room.Manager.Room, room.Code)
					room.Manager.Mutex.Unlock()
					if err := config.DeleteRoom(room.Code); err != nil {
						log.Println(err)
					}
					room.Hub.DoneOnce.Do(func() {
						close(room.Hub.Done)
					})

					log.Println("deleted room:", room.Code)
				}()
			}

		case message := <-h.Broadcast:
			// Parse incoming message to update state
			var wrapper Syncwrapper
			messageData := message.MessageVideo
			if messageData != nil {
				if err := json.Unmarshal(messageData, &wrapper); err == nil {
					if wrapper.MessageVideo != nil {
						msg := wrapper.MessageVideo
						log.Println("VideoURl Hub:", h.VideoURL)
						<-h.StateMutex
						switch msg.Type {
						case "play":
							h.IsPlaying = true
							h.PlayerTime = msg.CurrentTime
							h.LastUpdatedAt = msg.LastUpdatedAt
						case "pause":
							h.IsPlaying = false
							h.PlayerTime = msg.CurrentTime
							h.LastUpdatedAt = msg.LastUpdatedAt
						case "seek":
							h.PlayerTime = msg.CurrentTime
							h.LastUpdatedAt = msg.LastUpdatedAt
						case "video_change":
							h.VideoURL = msg.VideoURL
							h.IsPlaying = false
							h.PlayerTime = 0
							fmt.Println("video_change now url", h.VideoURL)
						default:
							fmt.Println("Unknown message type:", msg.Type)
						}
						updateRedisState := config.RoomState{
							RoomCode:      message.Sender.Room.Code,
							PlayerTime:    h.PlayerTime,
							IsPlaying:     h.IsPlaying,
							VideoURL:      h.VideoURL,
							LastUpdatedAt: h.LastUpdatedAt,
						}
						config.SaveRoom(updateRedisState)
						fmt.Println("Hub Last Data:", h.PlayerTime, h.LastUpdatedAt, h.IsPlaying, h.VideoURL, updateRedisState.VideoURL)
						h.StateMutex <- true
					}
				}

				// Broadcast to all clients EXCEPT sender
				broadcasttoclient(h, message, messageData, false)
			}
			if messageData = message.MessageChat; messageData != nil {
				broadcasttoclient(h, message, messageData, true)
			}
			if messageData = message.HeartBeat; messageData != nil {
				broadcasttoclient(h, message, messageData, true)
			}

		}
	}
}

func broadcasttoclient(h *Hub, message BroadcastMessage, messageData []byte, isClient bool) {
	for client := range h.Clients {
		// Skip the sender

		if !isClient && client == message.Sender {
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
