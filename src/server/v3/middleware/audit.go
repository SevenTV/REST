package middleware

import (
	"fmt"

	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
)

func Audit(gCtx global.Context) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		go func() {
			for ev := range ctx.Lifecycle.Listen(ctx) {
				if ev.Event == rest.LifecyclePhaseCompleted {
					fmt.Println("Should write audit logs here")
				}
			}
		}()
		return nil
	}
}
