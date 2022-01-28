package rest

import (
	"encoding/json"

	"github.com/SevenTV/Common/errors"
	"github.com/valyala/fasthttp"
)

type Ctx struct {
	*fasthttp.RequestCtx
}

type APIError = errors.APIError

func (c *Ctx) JSON(v interface{}) APIError {
	b, err := json.Marshal(v)
	if err != nil {
		c.SetStatusCode(InternalServerError)
		return errors.ErrInternalServerError().
			SetDetail("JSON Parsing Failed").
			SetFields(errors.Fields{"JSON_ERROR": err.Error()})
	}

	c.SetContentType("application/json")
	c.SetBody(b)
	return nil
}
