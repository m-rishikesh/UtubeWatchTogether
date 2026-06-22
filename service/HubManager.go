package service

import (
	"fmt"
	"gotth/config"
	"log"
	"math/rand"
	"sync"
	"time"
)

type HubManager struct {
	Room  map[string]*Room
	Mutex *sync.RWMutex
}

type Room struct {
	Code    string `json:"code"`
	Hub     *Hub   `json:"hub omitempty"`
	Clients map[string]*RTCClient
	Mu      sync.RWMutex
	Manager *HubManager
}

func generateCode() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = letters[rand.Intn(len(letters))]
	}
	return string(code)
}

func (hm *HubManager) RoomCleaner() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		hm.Mutex.Lock()

		for code, room := range hm.Room {

			room.Mu.RLock()
			empty := room.Hub.ClientSize() == 0
			room.Mu.RUnlock()

			if empty {
				delete(hm.Room, code)
				err := config.DeleteRoom(code)
				if err != nil {
					log.Println("failed to delete from redis cache")
				}
				fmt.Println("Deleted room:", code)
			}
		}

		hm.Mutex.Unlock()
	}
}

func (hm *HubManager) CreateRoom() *Room {
	var code string
	for {
		code = generateCode()
		exists, _ := config.RoomExists(code)
		if !exists {
			break
		}
	}
	hub := NewHub()

	room := &Room{Code: code,
		Hub:     hub,
		Clients: make(map[string]*RTCClient),
		Mu:      sync.RWMutex{},
		Manager: hm,
	}
	hm.Mutex.Lock()
	hm.Room[code] = room
	hm.Mutex.Unlock()
	err := config.SaveRoom(config.RoomState{
		RoomCode:   code,
		VideoURL:   "",
		IsPlaying:  room.Hub.IsPlaying,
		PlayerTime: room.Hub.PlayerTime,
	})
	if err != nil {
		log.Println("failed to save the room in redis")
	}
	go hub.Run()
	return room
}

func (hm *HubManager) FindRoom(code string) *Room {
	hm.Mutex.RLock()
	room, ok := hm.Room[code]
	hm.Mutex.RUnlock()

	if ok {
		fmt.Println(room)
		return room
	}

	roomstate, err := config.GetRoom(code)
	if err != nil || roomstate == nil {
		return nil
	}

	hub := NewHub()

	hub.PlayerTime = roomstate.PlayerTime
	hub.IsPlaying = roomstate.IsPlaying

	room = &Room{
		Code:    roomstate.RoomCode,
		Hub:     hub,
		Clients: map[string]*RTCClient{},
		Mu:      sync.RWMutex{},
		Manager: hm,
	}

	hm.Mutex.Lock()

	if existing, ok := hm.Room[code]; ok {
		hm.Mutex.Unlock()
		return existing
	}
	log.Println("adding from redis to the memory")
	hm.Room[code] = room
	hm.Mutex.Unlock()
	go hub.Run()

	return room
}
