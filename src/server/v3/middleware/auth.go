package middleware

import (
	"fmt"

	"github.com/SevenTV/REST/src/server/rest"
)

func Auth() rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		fmt.Println("This is middleware")

		return nil
	}
}
