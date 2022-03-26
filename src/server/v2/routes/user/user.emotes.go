package user

import (
	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v2/model"
)

type emotes struct {
	Ctx global.Context
}

func NewEmotes(gCtx global.Context) rest.Route {
	return &emotes{gCtx}
}

func (*emotes) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/{user}/emotes",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

// Get Channel Emotes
// @Summary Get Channel Emotes
// @Description List the channel emotes of a user
// @Tags users,emotes
// @Param user path string false "User ID, Twitch ID or Twitch Login"
// @Produce json
// @Success 200 {array} model.Emote
// @Router /users/{user}/emotes [get]
func (r *emotes) Handler(ctx *rest.Ctx) errors.APIError {
	return ctx.JSON(rest.OK, []*model.Emote{})
}
