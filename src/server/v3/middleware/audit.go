package middleware

import (
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
)

func Audit(gCtx global.Context) rest.Middleware {
	return func(ctx *rest.Ctx) rest.APIError {
		go func() {
			for ev := range ctx.Lifecycle.Listen(ctx) {
				if ev.Event == rest.LifecyclePhaseCompleted {
					// TODO: Write Audit Logs here
					{
					}
				}
			}
		}()
		return nil
	}
}
