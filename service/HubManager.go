package service

import (
	"fmt"
	"math/rand"
	"sync"
)

type HubManager struct {
	Room  map[string]*Room
	Mutex *sync.RWMutex
}

type Room struct {
	Code string `json:"code"`
	Hub  *Hub   `json:"hub omitempty"`
}

func generateCode() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = letters[rand.Intn(len(letters))]
	}
	return string(code)
}

func (hm *HubManager) CreateRoom() *Room {
	code := generateCode()

	hub := NewHub()

	room := &Room{Code: code, Hub: hub}

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
