package helpers

import (
	"io"

	"github.com/gofiber/fiber/v2"
)

type ResponseBuilder struct {
	c       *fiber.Ctx
	Status  HttpStatusCode
	Message string
}

// HttpResponse: Get a ResponseBuilder, which defaults to status 200 OK
func HttpResponse(c *fiber.Ctx) *ResponseBuilder {
	return &ResponseBuilder{c, 200, ""}
}

// SetStatus: Change the response status for the request
func (rb *ResponseBuilder) SetStatus(status HttpStatusCode) *ResponseBuilder {
	rb.Status = status
	return rb
}

func (rb *ResponseBuilder) SetMessage(msg string) *ResponseBuilder {
	rb.Message = msg
	return rb
}

func (rb *ResponseBuilder) Send(b []byte) error {
	return rb.c.Status(int(rb.Status)).Send(b)
}

func (rb *ResponseBuilder) SendStatus() error {
	return rb.c.Status(int(rb.Status)).SendStatus(int(rb.Status))
}

func (rb *ResponseBuilder) SendString(s string) error {
	return rb.c.Status(int(rb.Status)).SendString(s)
}

func (rb *ResponseBuilder) SendStream(stream io.Reader, size ...int) error {
	return rb.c.Status(int(rb.Status)).SendStream(stream)
}

func (rb *ResponseBuilder) SendAsError() error {
	return rb.c.Status(int(rb.Status)).JSON(&fiber.Map{
		"status":  rb.Status,
		"message": rb.Message,
	})
}

type HttpStatusCode int

const (
	// 1xx Informational
	HttpStatusCodeContinue          HttpStatusCode = 100
	HttpStatusCodeSwitchingProtocol HttpStatusCode = 101
	HttpStatusCodeProcessing        HttpStatusCode = 102
	HttpStatusCodeEarlyHints        HttpStatusCode = 103

	// 2xx Successful
	HttpStatusCodeOK                          HttpStatusCode = 200
	HttpStatusCodeCreated                     HttpStatusCode = 201
	HttpStatusCodeAccepted                    HttpStatusCode = 202
	HttpStatusCodeNonAuthoritativeInformation HttpStatusCode = 203
	HttpStatusCodeNoContent                   HttpStatusCode = 204
	HttpStatusCodeResetContent                HttpStatusCode = 205
	HttpStatusCodePartialContent              HttpStatusCode = 206
	HttpStatusCodeMultiStatus                 HttpStatusCode = 207
	HttpStatusCodeAlreadyReported             HttpStatusCode = 208
	HttpStatusCodeIMUsed                      HttpStatusCode = 226

	// 3xx Redirections
	HttpStatusCodeMultipleChoice    HttpStatusCode = 300
	HttpStatusCodeMovedPermanently  HttpStatusCode = 301
	HttpStatusCodeFound             HttpStatusCode = 302
	HttpStatusCodeSeeOther          HttpStatusCode = 303
	HttpStatusCodeNotModified       HttpStatusCode = 304
	HttpStatusCodeTemporaryRedirect HttpStatusCode = 307
	HttpStatusCodePermanentRedirect HttpStatusCode = 308

	// 4xx Client Errors
	HttpStatusCodeBadRequest                  HttpStatusCode = 400
	HttpStatusCodeUnauthorized                HttpStatusCode = 401
	HttpStatusCodePaymentRequired             HttpStatusCode = 402
	HttpStatusCodeForbidden                   HttpStatusCode = 403
	HttpStatusCodeNotFound                    HttpStatusCode = 404
	HttpStatusCodeMethodNotAllowed            HttpStatusCode = 405
	HttpStatusCodeNotAcceptable               HttpStatusCode = 406
	HttpStatusCodeProxyAuthenticationRequired HttpStatusCode = 407
	HttpStatusCodeRequestTimeout              HttpStatusCode = 408
	HttpStatusCodeConflict                    HttpStatusCode = 409
	HttpStatusCodeGone                        HttpStatusCode = 410
	HttpStatusCodeLengthRequired              HttpStatusCode = 411
	HttpStatusCodePreconditionFailed          HttpStatusCode = 412
	HttpStatusCodePayloadTooLarge             HttpStatusCode = 413
	HttpStatusCodeURITooLong                  HttpStatusCode = 414
	HttpStatusCodeUnsupportedMediaType        HttpStatusCode = 415
	HttpStatusCodeRangeNotSatisfiable         HttpStatusCode = 416
	HttpStatusCodeExpectationFailed           HttpStatusCode = 417
	HttpStatusCodeImATeapot                   HttpStatusCode = 418
	HttpStatusCodeMisdirectedRequest          HttpStatusCode = 421
	HttpStatusCodeUnprocessableEntity         HttpStatusCode = 422
	HttpStatusCodeLocked                      HttpStatusCode = 423
	HttpStatusCodeFailedDependency            HttpStatusCode = 424
	HttpStatusCodeTooEarly                    HttpStatusCode = 425
	HttpStatusCodeUpgradeRequired             HttpStatusCode = 426
	HttpStatusCodePreconditionRequired        HttpStatusCode = 428
	HttpStatusCodeTooManyRequests             HttpStatusCode = 429
	HttpStatusCodeRequestHeaderFieldsTooLarge HttpStatusCode = 431
	HttpStatusCodeUnavailableForLegalReasons  HttpStatusCode = 451

	// 5xx Server Errors
	HttpStatusCodeInternalServerError           HttpStatusCode = 500
	HttpStatusCodeNotImplemented                HttpStatusCode = 501
	HttpStatusCodeBadGateway                    HttpStatusCode = 502
	HttpStatusCodeServiceUnavailable            HttpStatusCode = 503
	HttpStatusCodeGatewayTimeout                HttpStatusCode = 504
	HttpStatusCodeHttpVersionNotSupported       HttpStatusCode = 505
	HttpStatusCodeVariantAlsoNegotiates         HttpStatusCode = 506
	HttpStatusCodeInsufficientStorage           HttpStatusCode = 507
	HttpStatusCodeLoopDetected                  HttpStatusCode = 508
	HttpStatusCodeNotExtended                   HttpStatusCode = 510
	HttpStatusCodeNetworkAuthenticationRequired HttpStatusCode = 511
)
