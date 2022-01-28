package rest

import (
	"github.com/fasthttp/router"
)

type Route interface {
	Config() RouteConfig
	Handler(ctx *Ctx) APIError
}

type Router = router.Router

type RouteConfig struct {
	URI        string
	Method     RouteMethod
	Children   []Route
	Middleware []Middleware
}

type RouteMethod string

const (
	GET     RouteMethod = "GET"
	POST    RouteMethod = "POST"
	PUT     RouteMethod = "PUT"
	PATCH   RouteMethod = "PATCH"
	DELETE  RouteMethod = "DELETE"
	OPTIONS RouteMethod = "OPTIONS"
)

type Middleware = func(ctx *Ctx) APIError

type APIErrorResponse struct {
	Status    int               `json:"status"`
	Error     string            `json:"error"`
	ErrorCode int               `json:"error_code"`
	Details   map[string]string `json:"details,omitempty"`
}

type HttpStatusCode int

const (
	// 1xx Informational
	Continue          = 100
	SwitchingProtocol = 101
	Processing        = 102
	EarlyHints        = 103

	// 2xx Successful
	OK                          = 200
	Created                     = 201
	Accepted                    = 202
	NonAuthoritativeInformation = 203
	NoContent                   = 204
	ResetContent                = 205
	PartialContent              = 206
	MultiStatus                 = 207
	AlreadyReported             = 208
	IMUsed                      = 226

	// 3xx Redirections
	MultipleChoice    = 300
	MovedPermanently  = 301
	Found             = 302
	SeeOther          = 303
	NotModified       = 304
	TemporaryRedirect = 307
	PermanentRedirect = 308

	// 4xx Client Errors
	BadRequest                  = 400
	Unauthorized                = 401
	PaymentRequired             = 402
	Forbidden                   = 403
	NotFound                    = 404
	MethodNotAllowed            = 405
	NotAcceptable               = 406
	ProxyAuthenticationRequired = 407
	RequestTimeout              = 408
	Conflict                    = 409
	Gone                        = 410
	LengthRequired              = 411
	PreconditionFailed          = 412
	PayloadTooLarge             = 413
	URITooLong                  = 414
	UnsupportedMediaType        = 415
	RangeNotSatisfiable         = 416
	ExpectationFailed           = 417
	ImATeapot                   = 418
	MisdirectedRequest          = 421
	UnprocessableEntity         = 422
	Locked                      = 423
	FailedDependency            = 424
	TooEarly                    = 425
	UpgradeRequired             = 426
	PreconditionRequired        = 428
	TooManyRequests             = 429
	RequestHeaderFieldsTooLarge = 431
	UnavailableForLegalReasons  = 451

	// 5xx Server Errors
	InternalServerError           = 500
	NotImplemented                = 501
	BadGateway                    = 502
	ServiceUnavailable            = 503
	GatewayTimeout                = 504
	HttpVersionNotSupported       = 505
	VariantAlsoNegotiates         = 506
	InsufficientStorage           = 507
	LoopDetected                  = 508
	NotExtended                   = 510
	NetworkAuthenticationRequired = 511
)

type Map map[string]interface{}
