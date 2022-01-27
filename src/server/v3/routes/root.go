package routes

import (
	"fmt"

	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v3/routes/auth"
	"github.com/valyala/fasthttp"
)

type Route struct{}

func New() rest.Route {
	return &Route{}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/v3",
		Method: rest.GET,
		Children: []rest.Route{
			auth.New(),
		},
	}
}

func (r *Route) Handler(ctx *fasthttp.RequestCtx) {
	fmt.Println("Root Route")
}
