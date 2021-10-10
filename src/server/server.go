package server

import (
	"net"

	"github.com/SevenTV/REST/src/configure"
	v3 "github.com/SevenTV/REST/src/server/v3"
	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	app      *fiber.App
	listener net.Listener
}

func New() *Server {
	l, err := net.Listen(configure.Config.GetString("http.type"), configure.Config.GetString("http.uri"))
	if err != nil {
		panic(err)
	}

	server := &Server{
		app: fiber.New(fiber.Config{
			DisablePreParseMultipartForm: true,
		}),
		listener: l,
	}

	server.app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Node-ID", configure.NodeName)
		return c.Next()
	})

	// v3
	v3.API(server.app)

	// 404
	server.app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(&fiber.Map{
			"status":  404,
			"message": "Not Found",
		})
	})

	go func() {
		err = server.app.Listener(server.listener)
		if err != nil {
			log.WithError(err).Fatal("failed to start http server")
		}
	}()

	return server
}

func (s *Server) Shutdown() error {
	return s.listener.Close()
}
