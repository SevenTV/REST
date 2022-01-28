package auth

import (
	"fmt"

	"github.com/SevenTV/REST/src/server/rest"
)

type Route struct{}

func New() rest.Route {
	return &Route{}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:      "/auth",
		Method:   rest.GET,
		Children: []rest.Route{},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	fmt.Println("Auth Route")

	return ctx.JSON(map[string]string{
		"foo": "bar",
	})
}
