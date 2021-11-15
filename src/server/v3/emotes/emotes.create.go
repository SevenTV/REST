package emotes

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/sirupsen/logrus"

	"github.com/SevenTV/Common/structures"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/helpers"
	"github.com/SevenTV/REST/src/server/middleware"
	"github.com/gofiber/fiber/v2"
)

const MAX_UPLOAD_SIZE = 2621440    // 2.5MB
const MAX_LOSSLESS_SIZE = 256000.0 // 250KB

func create(gCtx global.Context, router fiber.Router) {
	router.Post(
		"/",
		middleware.Auth(gCtx),
		func(c *fiber.Ctx) error {
			ctx := c.Context()
			ctx.SetContentType("application/json")

			// Get actor
			actor, ok := c.Locals("user").(*structures.User)
			if !ok {
				return helpers.HttpResponse(c).
					SetStatus(helpers.HttpStatusCodeUnauthorized).
					SetMessage("Authentication Required").
					SendAsError()
			}

			req := c.Request()
			if !req.IsBodyStream() {
				return helpers.HttpResponse(c).
					SetStatus(helpers.HttpStatusCodeBadRequest).
					SetMessage("Not A File Stream").
					SendAsError()
			}

			// Stream incoming file
			mr := multipart.NewReader(ctx.RequestBodyStream(), utils.B2S(req.Header.MultipartFormBoundary()))
			eb := structures.EmoteBuilder{
				Update: map[string]interface{}{},
				Emote:  &structures.Emote{},
			}

			var (
				contentType string
				file        *bytes.Reader
				fileLength  int
			)
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				} else if err != nil {
					logrus.WithError(err).Error("multipart")
					break
				}
				if part.FormName() != "file" {
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SetMessage("Invalid Form Data").SendAsError()
				}

				b, err := io.ReadAll(part)
				if err != nil {
					logrus.WithError(err).Warn("io, ReadAll")
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SetMessage("File Unreadable").SendAsError()
				}
				if len(b) > MAX_UPLOAD_SIZE {
					return helpers.HttpResponse(c).
						SetStatus(helpers.HttpStatusCodePayloadTooLarge).
						SetMessage("Input File Too Large. Must be <2.5MB").
						SendAsError()
				}

				fileLength = len(b)
				file = bytes.NewReader(b)
				contentType = part.Header.Get("Content-Type")
				if contentType == "" {
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SetMessage("Unknown Content Type").SendAsError()
				}
			}

			fmt.Println(actor, eb, file, fileLength)
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeOK).SendString("")
		},
	)
}

type EmoteSize struct {
	Name    string
	Width   int32
	Height  int32
	Quality int32
}

var emoteSizes = []EmoteSize{
	{
		Name:   "1x",
		Width:  96,
		Height: 32,
	},
	{
		Name:   "2x",
		Width:  192,
		Height: 64,
	},
	{
		Name:   "3x",
		Width:  288,
		Height: 96,
	},
	{
		Name:   "4x",
		Width:  384,
		Height: 128,
	},
}
