package server

import (
	"github.com/SevenTV/REST/src/server/types"
	v3 "github.com/SevenTV/REST/src/server/v3"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func (s *HttpServer) V3() {
	s.traverseRoutes(v3.API(s.Ctx, s.router), nil)
}

func (s *HttpServer) traverseRoutes(r types.Route, parent types.Route) {
	c := r.Config()

	var caller func(path string, handler fasthttp.RequestHandler)
	switch c.Method {
	case types.MethodGET:
		caller = s.router.GET
	case types.MethodPOST:
		caller = s.router.POST
	case types.MethodPUT:
		caller = s.router.PUT
	case types.MethodPATCH:
		caller = s.router.PATCH
	case types.MethodDELETE:
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
