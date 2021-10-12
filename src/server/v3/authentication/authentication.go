package authentication

import "github.com/gofiber/fiber/v2"

func Authentication(router fiber.Router) {
	group := router.Group("/auth")

	group.Get("")
}
