package service

import (
	"fmt"
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
				fmt.Println("Deleted room:", code)
			}
		}

		hm.Mutex.Unlock()
	}
}

func (hm *HubManager) CreateRoom() *Room {
	code := generateCode()

	hub := NewHub()

	room := &Room{Code: code, Hub: hub, Clients: make(map[string]*RTCClient), Mu: sync.RWMutex{}}

	hm.Room[code] = room

	go hub.Run()
	return room
}

func (hm *HubManager) FindRoom(code string) *Room {
	hm.Mutex.RLock()
	defer hm.Mutex.RUnlock()

	room, ok := hm.Room[code]
	if !ok {
		return nil
	}

	fmt.Println(room)

	return room
}
