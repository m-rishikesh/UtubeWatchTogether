package handlers

import (
	"gotth/cmd/Templ/components"

	"github.com/labstack/echo/v5"
)

func JoinRoomForm(c *echo.Context) error {
	join_component := components.JoinRoom()
	return render(c, join_component)
}
