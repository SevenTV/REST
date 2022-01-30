package emotes

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

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/emotes",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrDontBeSilly()
}
