package server

import (
	"encoding/json"
	"strings"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	v3 "github.com/SevenTV/REST/src/server/v3"
	"github.com/fasthttp/router"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func (s *HttpServer) V3(gCtx global.Context) {
	s.traverseRoutes(v3.API(gCtx, s.router), s.router)
}

func (s *HttpServer) SetupHandlers() {
	// Handle Not Found
	s.router.NotFound = s.getErrorHandler(
		rest.NotFound,
		errors.ErrUnknownRoute().SetFields(errors.Fields{
			"message": "The API endpoint requested does not exist",
		}),
	)

	// Handle P A N I C
	s.router.PanicHandler = func(ctx *fasthttp.RequestCtx, i interface{}) {
		err := "Uh oh. Something went horribly wrong"
		switch x := i.(type) {
		case error:
			err += ": " + x.Error()
		case string:
			err += ": " + x
		}
		s.getErrorHandler(
			rest.InternalServerError,
			errors.ErrInternalServerError().SetFields(errors.Fields{
				"panic": err,
			}),
		)(ctx)
	}
}

func (s *HttpServer) traverseRoutes(r rest.Route, parentGroup Router) {
	c := r.Config()

	// Compose the full request URI (prefixing with parent, if any)
	routable := parentGroup
	group := routable.Group(c.URI)
	l := logrus.WithFields(logrus.Fields{
		"group":  group,
		"method": c.Method,
	})

	// Handle requests
	group.Handle(string(c.Method), "", func(ctx *fasthttp.RequestCtx) {
		rctx := &rest.Ctx{
			Lifecycle:  &rest.Lifecycle{},
			RequestCtx: ctx,
		}
		handlers := make([]rest.Middleware, len(c.Middleware)+1)
		for i, mw := range c.Middleware {
			handlers[i] = mw
		}
		handlers[len(handlers)-1] = r.Handler

		for i, h := range handlers {
			if i == len(handlers)-1 {
				// emit "started" lifecycle event after middlewares
				rctx.Lifecycle.Write(rest.LifecyclePhaseStarted, nil)
			}
			if err := h(rctx); err != nil {
				// If the request handler returned an error
				// we will format it into standard API error response
				resp := &rest.APIErrorResponse{
					Status:    ctx.Response.StatusCode(),
					Error:     strings.Title(err.Message()),
					ErrorCode: err.Code(),
					Details:   err.GetFields(),
				}

				b, _ := json.Marshal(resp)
				ctx.SetContentType("application/json")
				ctx.SetBody(b)
				return
			}
		}
		// emit "completed" lifecycle event once all handlers have completed
		rctx.Lifecycle.Write(rest.LifecyclePhaseCompleted, nil)
	})
	l.Debug("Route registered")

	// activate child routes
	for _, child := range c.Children {
		s.traverseRoutes(child, group)
	}
}

func (s *HttpServer) getErrorHandler(status rest.HttpStatusCode, err rest.APIError) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		b, _ := json.Marshal(&rest.APIErrorResponse{
			Status:    int(status),
			Error:     strings.Title(err.Message()),
			ErrorCode: err.Code(),
			Details:   err.GetFields(),
		})
		ctx.SetContentType("application/json")
		ctx.SetBody(b)
	}
}

type Router interface {
	ANY(path string, handler fasthttp.RequestHandler)
	CONNECT(path string, handler fasthttp.RequestHandler)
	DELETE(path string, handler fasthttp.RequestHandler)
	GET(path string, handler fasthttp.RequestHandler)
	Group(path string) *router.Group
	HEAD(path string, handler fasthttp.RequestHandler)
	Handle(method, path string, handler fasthttp.RequestHandler)
	OPTIONS(path string, handler fasthttp.RequestHandler)
	PATCH(path string, handler fasthttp.RequestHandler)
	POST(path string, handler fasthttp.RequestHandler)
	PUT(path string, handler fasthttp.RequestHandler)
	ServeFiles(path string, rootPath string)
	ServeFilesCustom(path string, fs *fasthttp.FS)
	TRACE(path string, handler fasthttp.RequestHandler)
}
