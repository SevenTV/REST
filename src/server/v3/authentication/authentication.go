package authentication

import (
	"encoding/json"

	"github.com/SevenTV/REST/src/global"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func Authentication(gCtx global.Context, router fiber.Router) {
	group := router.Group("/auth")
	twitch(gCtx, group)

	group.Get("/sign", func(c *fiber.Ctx) error {
		claim, err := json.Marshal(map[string]interface{}{
			"u": "XD",
			"x": []int{1, 2, 3},
		})
		if err != nil {
			logrus.WithError(err).Error("json")
			return c.SendStatus(500)
		}

		token, err := gCtx.Inst().Auth.Sign(gCtx.Config().NodeName, claim)
		if err != nil {
			logrus.WithError(err).Error("sign")
			return c.SendStatus(500)
		}

		return c.SendString(token)
	})

	group.Get("/verify", func(c *fiber.Ctx) error {
		t := c.Query("token")

		token, err := gCtx.Inst().Auth.Verify(t)
		if err != nil {
			logrus.WithError(err).Error("verify")
			return c.SendStatus(500)
		}

		return c.SendStatus(200)
	})
}
