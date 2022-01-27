package auth

import (
	"fmt"

	"github.com/SevenTV/REST/src/server/rest"
	"github.com/valyala/fasthttp"
)

type Route struct{}

func New() rest.Route {
	return &Route{}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/auth",
		Method:   rest.GET,
		Children: []rest.Route{},
	}
}

func (r *Route) Handler(ctx *fasthttp.RequestCtx) {
	fmt.Println("Auth Route")
}
