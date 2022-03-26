package user

import (
	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

// Config implements rest.Route
func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/users",
		Method: rest.GET,
		Children: []rest.Route{
			NewEmotes(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

// Handler implements rest.Route
func (*Route) Handler(ctx *rest.Ctx) errors.APIError {
	return ctx.JSON(rest.OK, []string{})
}
