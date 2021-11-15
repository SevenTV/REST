package server

import (
	"net"
	"time"

	"github.com/SevenTV/REST/src/global"
	v3 "github.com/SevenTV/REST/src/server/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func New(gCtx global.Context) <-chan struct{} {
	ln, err := net.Listen(gCtx.Config().Http.Type, gCtx.Config().Http.URI)
	if err != nil {
		logrus.WithError(err).Fatal("failed to start http server")
	}

	app := fiber.New(fiber.Config{
		BodyLimit:                    2e16,
		StreamRequestBody:            true,
		DisableStartupMessage:        true,
		DisablePreParseMultipartForm: true,
		DisableKeepalive:             true,
		ReadTimeout:                  time.Second * 10,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Node-ID", gCtx.Config().NodeName)
		return c.Next()
	})

	// v3
	v3.API(gCtx, app.Group("/v3"))

	// 404
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(&fiber.Map{
			"status":  404,
			"message": "Not Found",
		})
	})

	go func() {
		err = app.Listener(ln)
		if err != nil {
			logrus.WithError(err).Fatal("failed to start http server")
		}
	}()

	done := make(chan struct{})

	go func() {
		<-gCtx.Done()
		_ = app.Shutdown()
		close(done)
	}()

	return done
}
