package authentication

import (
	"encoding/json"
	"fmt"

	"github.com/SevenTV/REST/src/auth"
	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

func Authentication(router fiber.Router) {
	group := router.Group("/auth")

	group.Get("/sign", func(c *fiber.Ctx) error {
		claim, err := json.Marshal(map[string]interface{}{
			"u": "XD",
			"x": []int{1, 2, 3},
		})
		if err != nil {
			log.WithError(err).Error("json")
			return c.SendStatus(500)
		}

		token, err := auth.ECDSA.Sign(claim)
		if err != nil {
			log.WithError(err).Error("sign")
			return c.SendStatus(500)
		}

		return c.SendString(token)
	})

	group.Get("/verify", func(c *fiber.Ctx) error {
		t := c.Query("token")

		token, err := auth.ECDSA.Verify(t)
		if err != nil {
			log.WithError(err).Error("verify")
			return c.SendStatus(500)
		}

		log.Info("Pog!")
		fmt.Println(token)
		return c.SendStatus(200)
	})
}
