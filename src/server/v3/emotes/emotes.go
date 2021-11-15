package emotes

import (
	"github.com/SevenTV/REST/src/global"
	"github.com/gofiber/fiber/v2"
)

func Emotes(gCtx global.Context, router fiber.Router) {
	group := router.Group("/emotes")

	create(gCtx, group)
}
