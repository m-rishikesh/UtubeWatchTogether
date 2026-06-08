package handlers

import (
	"fmt"
	"gotth/cmd/Templ/components"
	"gotth/service"
	"net/http"

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
	if code == "" {
		fmt.Println("code is empty")
	} else {
		fmt.Println("code", code)
	}
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
