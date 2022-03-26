package emotes

import (
	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/loaders"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v2/model"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

// Config implements rest.Route
func (*Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/emotes/{emote}",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []func(ctx *rest.Ctx) errors.APIError{},
	}
}

// Handler implements rest.Route
func (r *Route) Handler(ctx *rest.Ctx) errors.APIError {
	emoteID, err := ctx.UserValue(rest.Key("emote")).ObjectID()
	if err != nil {
		return errors.From(err)
	}

	emote, err := loaders.For(ctx).EmoteByID.Load(emoteID)
	if err != nil {
		return errors.From(err)
	}

	return ctx.JSON(rest.OK, model.NewEmote(r.Ctx, emote))
}
