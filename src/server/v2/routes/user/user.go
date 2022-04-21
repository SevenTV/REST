package user

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

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/users/{user}",
		Method: rest.GET,
		Children: []rest.Route{
			newEmotes(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

// Get User
// @Summary Get User
// @Description Finds a user by its ID, Username or Twitch ID
// @Tags users
// @Param user path string false "User ID, Username or Twitch ID"
// @Produce json
// @Success 200 {object} model.User
// @Router /users/{user} [get]
func (*Route) Handler(ctx *rest.Ctx) errors.APIError {
	key, _ := ctx.UserValue("user").String()
	user, err := loaders.For(ctx).UserByIdentifier.Load(key)
	if err != nil {
		return errors.From(err)
	}
	if user == nil || user.ID.IsZero() {
		return errors.ErrUnknownUser()
	}
	return ctx.JSON(rest.OK, model.NewUser(*user))
}
