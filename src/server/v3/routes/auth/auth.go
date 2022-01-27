package auth

import (
	"fmt"

	"github.com/SevenTV/REST/src/server/types"
	"github.com/valyala/fasthttp"
)

type Route struct{}

func New() types.Route {
	return &Route{}
}

func (r *Route) Config() types.RouteConfig {
	return types.RouteConfig{
		URI:      "/auth",
		Method:   types.MethodGET,
		Children: []types.Route{},
	}
}

func (r *Route) Handler(ctx *fasthttp.RequestCtx) {
	fmt.Println("Auth Route")
}
