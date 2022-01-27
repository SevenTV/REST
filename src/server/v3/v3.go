package v3

import (
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/types"
	"github.com/SevenTV/REST/src/server/v3/routes"
)

func API(gCtx global.Context, router *types.Router) types.Route {
	return routes.New()
}
