package handlers

import (
	"gotth/cmd/Templ/components"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v5"
)

func render(c *echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response())
}

func Home(c *echo.Context) error {
	components := components.VideoPlayer()
	return render(c, components)
}

func RandomNumberGenerator(c *echo.Context) error {
	randomnumber := rand.Int()
	return c.HTML(http.StatusOK, strconv.Itoa(randomnumber))
}
