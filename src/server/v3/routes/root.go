package routes

import (
	"fmt"

	"github.com/SevenTV/REST/src/server/types"
	"github.com/SevenTV/REST/src/server/v3/routes/auth"
	"github.com/valyala/fasthttp"
)

type Route struct{}

func New() types.Route {
	return &Route{}
}

func (r *Route) Config() types.RouteConfig {
	return types.RouteConfig{
		URI:    "/v3",
		Method: types.MethodGET,
		Children: []types.Route{
			auth.New(),
		},
	}
}

func (r *Route) Handler(ctx *fasthttp.RequestCtx) {
	fmt.Println("Root Route")
}
