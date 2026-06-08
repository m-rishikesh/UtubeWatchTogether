package handlers

import (
	"gotth/cmd/Templ/components"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v5"
)

func render(c *echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response())
}

func Home(c *echo.Context) error {
	components := components.Home()
	return render(c, components)
}
