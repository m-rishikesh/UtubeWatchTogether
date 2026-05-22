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

func WebsktHandler(c *echo.Context) error {
	// create a new client everytime this handler being called

	conn, error := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if error != nil {
		log.Println(error)
		return error
	}
	client := &service.Client{Conn: conn, Send: make(chan []byte, 256)}
	Hub.Register <- client
	log.Println("Connected:", client)
	go client.SendToHub(Hub)
	go client.ReceiveFromHub(Hub)

	return nil
}
