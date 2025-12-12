package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func BasicAuthMiddleware(username, password string) echo.MiddlewareFunc {
	return middleware.BasicAuth(func(user, pass string, c echo.Context) (bool, error) {
		if user == username && pass == password {
			return true, nil
		}
		return false, nil
	})
}
