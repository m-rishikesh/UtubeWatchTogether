package main

import (
	"gotth/handlers"
	"log"

	"github.com/labstack/echo/v5"
)

func main() {
	go handlers.Hub.Run()
	e := echo.New()
	e.Static("/static", "../../static")

	e.GET("/", handlers.Home)
	e.GET("/ws", handlers.WebsktHandler)

	log.Fatal(e.Start(":8080"))
}
