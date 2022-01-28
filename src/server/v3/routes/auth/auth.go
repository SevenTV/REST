package auth

import (
	"fmt"

	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v3/middleware"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/auth",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.Auth(r.Ctx),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	fmt.Println("Auth Route")

	return ctx.JSON(rest.OK, map[string]string{
		"foo": "bar",
	})
}
