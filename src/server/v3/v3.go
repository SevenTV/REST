package v3

import (
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v3/routes"
)

func API(gCtx global.Context, router *rest.Router) rest.Route {
	return routes.New()
}
