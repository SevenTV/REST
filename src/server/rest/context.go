package rest

import "github.com/valyala/fasthttp"

type Ctx struct {
	*fasthttp.RequestCtx
}
