package handlers

import (
	"fmt"
	"gotth/cmd/Templ/components"
	"gotth/service"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

func CreateRoomHandler(c *echo.Context, hm *service.HubManager) error {
	room := hm.CreateRoom()

	// Create a Room struct for the template
	roomData := components.Room{
		Code: room.Code,
	}

	// Render the RoomActive template
	component := components.RoomActive(roomData)
	return component.Render(c.Request().Context(), c.Response())
}

func JoinRoomHandler(c *echo.Context, hm *service.HubManager) error {
	code := c.Request().FormValue("room_code")
	code = strings.ToUpper(code)
	if code == "" {
		fmt.Println("code is empty")
	} else {
		fmt.Println("code", code)
	}
	log.Println("joinroom calling findroom")
	room := hm.FindRoom(code)
	fmt.Println("room:", room)
	if room == nil {
		http.Error(
			c.Response(),
			"Room Not Found",
			http.StatusNotFound,
		)
		return nil
	}

	if room.Hub.ClientSize() >= 4 {
		http.Error(
			c.Response(),
			"Room Full",
			http.StatusForbidden,
		)
		return nil
	}

	// Create a Room struct for the template
	roomData := components.Room{
		Code: room.Code,
	}

	// Render the RoomActive template
	component := components.RoomActive(roomData)
	return component.Render(c.Request().Context(), c.Response())
}

func AutoJoinHandler(c *echo.Context, hm *service.HubManager) error {

	code := c.QueryParam("code")
	room := hm.FindRoom(code)
	if room != nil {
		return components.RoomActive(components.Room{Code: code}).Render(c.Request().Context(), c.Response())
	}
	return nil

}
