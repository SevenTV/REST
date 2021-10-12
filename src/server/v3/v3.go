package v3

import (
	"github.com/SevenTV/REST/src/server/v3/authentication"
	"github.com/gofiber/fiber/v2"
)

func API(router fiber.Router) {
	authentication.Authentication(router)
}
