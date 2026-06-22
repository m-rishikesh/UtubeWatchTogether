package main

import (
	"gotth/handlers"
	"gotth/service"
	"log"
	"sync"

	"github.com/labstack/echo/v5"
)

func main() {
	hm := &service.HubManager{Room: make(map[string]*service.Room), Mutex: &sync.RWMutex{}}
	// go hm.RoomCleaner()
	e := echo.New()
	e.Static("/static", "./static")

	e.GET("/", handlers.Home)
	e.GET("/ws", func(c *echo.Context) error {
		return handlers.WebsktHandler(c, hm)
	})
	e.GET("/ws/video", func(c *echo.Context) error {
		return handlers.WsEchoHandler(c, hm)
	})
	e.GET("/join-room-form", handlers.JoinRoomForm)

	e.POST("/create-room", func(c *echo.Context) error {
		return handlers.CreateRoomHandler(c, hm)
	})
	e.POST("/join-room", func(c *echo.Context) error {
		return handlers.JoinRoomHandler(c, hm)
	})

	log.Fatal(e.Start(":8080"))
}
