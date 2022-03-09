package auth

import (
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
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	ctx.Redirect("/v3/auth/twitch?old=true", int(rest.Found))
	return nil
}
