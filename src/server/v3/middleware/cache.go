package middleware

import (
	"fmt"
	"strings"

	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
)

func SetCacheControl(gCtx global.Context, maxAge int, args []string) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		ctx.Response.Header.Set("Cache-Control", fmt.Sprintf(
			"max-age=%d%s %s",
			maxAge,
			utils.Ternary(len(args) > 0, ",", "").(string),
			strings.Join(args, ", "),
		))

		return nil
	}
}
