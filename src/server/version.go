package server

import (
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	v3 "github.com/SevenTV/REST/src/server/v3"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func (s *HttpServer) V3(gCtx global.Context) {
	s.traverseRoutes(v3.API(gCtx, s.router), nil)
}

func (s *HttpServer) traverseRoutes(r rest.Route, parent rest.Route) {
	c := r.Config()

	var caller func(path string, handler fasthttp.RequestHandler)
	switch c.Method {
	case rest.GET:
		caller = s.router.GET
	case rest.POST:
		caller = s.router.POST
	case rest.PUT:
		caller = s.router.PUT
	case rest.PATCH:
		caller = s.router.PATCH
	case rest.DELETE:
		caller = s.router.DELETE
	}

	if caller == nil {
		logrus.Errorf("Unknown Method: %s", c.Method)
		return
	}

	uri := ""
	if parent != nil {
		uri = parent.Config().URI
	}
	uri += c.URI
	logrus.WithFields(logrus.Fields{
		"uri":    uri,
		"method": c.Method,
	}).Debug("Route Registered")
	caller(uri, r.Handler)

	// activate child routes
	for _, child := range c.Children {
		s.traverseRoutes(child, r)
	}
}
