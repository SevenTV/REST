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

func (r *Route) Handler(ctx *rest.Ctx) {
	fmt.Println("Auth Route")
}
