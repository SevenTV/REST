package v3

import (
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/v3/authentication"
	"github.com/gofiber/fiber/v2"
)

func API(gCtx global.Context, router fiber.Router) {
	authentication.Authentication(gCtx, router)
}
