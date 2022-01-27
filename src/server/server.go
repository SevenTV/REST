package server

import (
	"net"
	"time"

	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/global"
	"github.com/fasthttp/router"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type HttpServer struct {
	listener net.Listener
	server   *fasthttp.Server
	router   *router.Router
}

// Start: set up the http server and begin listening on the configured port
func (s *HttpServer) Start(gCtx global.Context) (<-chan struct{}, error) {
	var err error
	s.listener, err = net.Listen(gCtx.Config().Http.Type, gCtx.Config().Http.URI)
	if err != nil {
		return nil, err
	}
	s.router = router.New()

	// Add versions
	s.V3(gCtx)

	s.server = &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			defer func() {
				l := logrus.WithFields(logrus.Fields{
					"status":   ctx.Response.StatusCode(),
					"duration": time.Since(start) / time.Millisecond,
					"path":     utils.B2S(ctx.Path()),
				})
				if err := recover(); err != nil {
					l.Error("panic in handler: ", err)
				} else {
					l.Info()
				}
			}()

			// Routing
			s.router.Handler(ctx)
		},
		ReadTimeout:                  time.Second * 600,
		MaxRequestBodySize:           2e16,
		DisableKeepalive:             true,
		DisablePreParseMultipartForm: true,
		LogAllErrors:                 true,
		StreamRequestBody:            true,
	}

	// Begin listening
	go func() {
		if err = s.server.Serve(s.listener); err != nil {
			logrus.WithError(err).Fatal("failed to start http server")
		}
	}()

	// Gracefully exit when the global context is canceled
	done := make(chan struct{})
	go func() {
		<-gCtx.Done()
		_ = s.server.Shutdown()
		close(done)
	}()

	return done, err
}

func New() HttpServer {
	return HttpServer{}
}
