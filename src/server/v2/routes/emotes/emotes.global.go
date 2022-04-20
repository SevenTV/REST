package emotes

import (
	"github.com/SevenTV/Common/errors"
	v2structures "github.com/SevenTV/Common/structures/v2"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v2/model"
	"github.com/SevenTV/REST/src/server/v3/middleware"
)

type globals struct {
	Ctx global.Context
}

func newGlobals(gCtx global.Context) rest.Route {
	return &globals{gCtx}
}

func (r *globals) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/global",
		Method:   rest.GET,
		Children: []rest.Route{},
		Middleware: []rest.Middleware{
			middleware.SetCacheControl(r.Ctx, 3600, nil),
		},
	}
}

// Get Global Emotes
// @Summary Get Globla Emotes
// @Description Lists active global emotes
// @Tags emotes
// @Produce json
// @Success 200 {array} model.Emote
// @Router /emotes/global [get]
func (r *globals) Handler(ctx *rest.Ctx) errors.APIError {
	es, err := r.Ctx.Inst().Query.GlobalEmoteSet(ctx)
	if err != nil {
		return errors.From(err)
	}

	result := make([]*model.Emote, len(es.Emotes))
	for i, ae := range es.Emotes {
		if ae.Emote == nil {
			continue
		}
		result[i] = model.NewEmote(r.Ctx, *ae.Emote)
		result[i].Visibility |= v2structures.EmoteVisibilityGlobal
	}
	return ctx.JSON(rest.OK, result)
}