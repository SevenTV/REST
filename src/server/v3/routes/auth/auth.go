package auth

import (
	"fmt"

	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/auth",
		Method: rest.GET,
		Children: []rest.Route{
			newTwitch(r.Ctx),
			newTwitchCallback(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	fmt.Println("Auth Route")

	return ctx.JSON(rest.OK, map[string]string{
		"foo": "bar",
	})
}
