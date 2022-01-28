package server

import (
	"encoding/json"
	"strings"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	v3 "github.com/SevenTV/REST/src/server/v3"
	"github.com/fasthttp/router"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func (s *HttpServer) V3(gCtx global.Context) {
	s.traverseRoutes(v3.API(gCtx, s.router), nil, nil)
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

func (s *HttpServer) traverseRoutes(r rest.Route, parent rest.Route, parentGroup *router.Group) {
	c := r.Config()

	// Compose the full request URI (prefixing with parent, if any)
	routable := utils.Ternary(parentGroup == nil, s.router.Group(""), parentGroup).(*router.Group)
	group := routable.Group(c.URI)
	l := logrus.WithFields(logrus.Fields{
		"group":  group,
		"method": c.Method,
	})

	// Handle requests
	group.Handle(string(c.Method), "", func(ctx *fasthttp.RequestCtx) {
		rctx := &rest.Ctx{RequestCtx: ctx}
		handlers := make([]rest.Middleware, len(c.Middleware)+1)
		for i, mw := range c.Middleware {
			handlers[i] = mw
		}
		handlers[len(handlers)-1] = r.Handler

		for _, h := range handlers {
			if err := h(rctx); err != nil {
				// If the request handler returned an error
				// we will format it into standard API error response
				resp := &rest.APIErrorResponse{
					Status:    ctx.Response.StatusCode(),
					Error:     err.Message(),
					ErrorCode: err.Code(),
					Details:   err.GetFields(),
				}

				b, _ := json.Marshal(resp)
				ctx.SetContentType("application/json")
				ctx.SetBody(b)
				return
			}
		}
	})
	l.Debug("Route registered")

	// activate child routes
	for _, child := range c.Children {
		s.traverseRoutes(child, r, group)
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
