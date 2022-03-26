package routes

import (
	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v2/routes/auth"
	"github.com/SevenTV/REST/src/server/v2/routes/cosmetics"
	"github.com/SevenTV/REST/src/server/v2/routes/emotes"
	"github.com/SevenTV/REST/src/server/v2/routes/user"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/v2",
		Method: rest.GET,
		Children: []rest.Route{
			auth.New(r.Ctx),
			user.New(r.Ctx),
			emotes.New(r.Ctx),
			cosmetics.New(r.Ctx),
		},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrUnknownRoute()
}
