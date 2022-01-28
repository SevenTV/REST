package server

import (
	"encoding/json"
	"strings"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	v3 "github.com/SevenTV/REST/src/server/v3"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func (s *HttpServer) V3(gCtx global.Context) {
	s.traverseRoutes(v3.API(gCtx, s.router), nil)
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

	s.router.GlobalOPTIONS = func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Access-Control-Max-Age", "7200")
	}
}

func (s *HttpServer) traverseRoutes(r rest.Route, parent rest.Route) {
	c := r.Config()

	// Compose the full request URI (prefixing with parent, if any)
	uri := ""
	if parent != nil {
		uri = parent.Config().URI
	}
	uri += c.URI
	l := logrus.WithFields(logrus.Fields{
		"uri":    uri,
		"method": c.Method,
	})

	// The route cannot already have been defined
	if s.hasRoute(uri) {
		l.Panic("Route already defined")
	}

	// Handle requests
	s.router.Handle(string(c.Method), uri, func(ctx *fasthttp.RequestCtx) {
		if err := r.Handler(&rest.Ctx{RequestCtx: ctx}); err != nil {
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
		}
	})
	s.addRoute(uri, &r)
	l.Debug("Route registered")

	// activate child routes
	for _, child := range c.Children {
		s.traverseRoutes(child, r)
	}
}

func (s *HttpServer) addRoute(k string, r *rest.Route) {
	s.routes[k] = r
}

func (s *HttpServer) hasRoute(k string) bool {
	_, ok := s.routes[k]

	return ok
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
