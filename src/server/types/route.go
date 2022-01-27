package types

import (
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

type Route interface {
	Config() RouteConfig
	Handler(ctx *fasthttp.RequestCtx)
}

type Router = router.Router

type RouteConfig struct {
	URI      string
	Method   RouteMethod
	Children []Route
}

type RouteMethod string

const (
	MethodGET     RouteMethod = "GET"
	MethodPOST    RouteMethod = "POST"
	MethodPUT     RouteMethod = "PUT"
	MethodPATCH   RouteMethod = "PATCH"
	MethodDELETE  RouteMethod = "DELETE"
	MethodOPTIONS RouteMethod = "OPTIONS"
)
