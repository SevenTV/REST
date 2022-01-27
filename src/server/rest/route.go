package rest

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
	GET     RouteMethod = "GET"
	POST    RouteMethod = "POST"
	PUT     RouteMethod = "PUT"
	PATCH   RouteMethod = "PATCH"
	DELETE  RouteMethod = "DELETE"
	OPTIONS RouteMethod = "OPTIONS"
)
