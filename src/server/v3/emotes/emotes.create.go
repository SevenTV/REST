package emotes

import (
	"bytes"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/aws"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/helpers"
	"github.com/SevenTV/REST/src/server/middleware"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/seventv/EmoteProcessor/src/containers"
	"github.com/seventv/EmoteProcessor/src/image"
	"github.com/seventv/EmoteProcessor/src/job"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	MAX_UPLOAD_SIZE   = 2621440  // 2.5MB
	MAX_LOSSLESS_SIZE = 256000.0 // 250KB
	MAX_FRAMES        = 750
	MAX_WIDTH         = 1000
	MAX_HEIGHT        = 1000
	MAX_TAGS          = 6
)

var (
	emoteNameRegex = regexp.MustCompile(`^[-_A-Za-z():0-9]{2,100}$`)
	emoteTagRegex  = regexp.MustCompile(`^[0-9a-z]{3,30}$`)
	webpMuxRegex   = regexp.MustCompile(`Canvas size: (\d+) x (\d+)(?:\n?.*){0,3}(?:Number of frames: (\d+))?`) // capture group 1: width, 2: height, 3: frame count or empty which means 1
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func create(gCtx global.Context, router fiber.Router) {
	router.Post(
		"/",
		middleware.Auth(gCtx),
		func(c *fiber.Ctx) error {
			ctx := c.Context()
			ctx.SetContentType("application/json")

			// Check RMQ status
			if gCtx.Inst().Rmq == nil {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeLocked).SetMessage("Emote Processing Service Unavailable").SendAsError()
			}

			// Get actor
			actor, ok := c.Locals("user").(*structures.User)
			if !ok {
				return helpers.HttpResponse(c).
					SetStatus(helpers.HttpStatusCodeUnauthorized).
					SetMessage("Authentication Required").
					SendAsError()
			}

			if !actor.HasPermission(structures.RolePermissionCreateEmote) {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeForbidden).SendString("Insufficient Privilege")
			}

			req := c.Request()
			var (
				name  string
				tags  []string
				flags structures.EmoteFlag
			)

			// these validations are all "free" as in we can do them before we download the file they try to upload.
			args := &CreateEmoteData{}
			if err := json.Unmarshal(req.Header.Peek("X-Emote-Data"), args); err != nil {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SetMessage(err.Error()).SendAsError()
			}

			// Validate: Name
			{
				if !emoteNameRegex.MatchString(args.Name) {
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SendString("Bad Emote Name")
				}
				name = args.Name
			}
			// Validate: Flags
			{
				if args.Flags != 0 {
					if utils.BitField.HasBits(int64(args.Flags), int64(structures.EmoteFlagsPrivate)) {
						flags |= structures.EmoteFlagsPrivate
					}
					if utils.BitField.HasBits(int64(args.Flags), int64(structures.EmoteFlagsZeroWidth)) {
						flags |= structures.EmoteFlagsZeroWidth
					}
				}
			}

			// Validate: Tags
			{
				uniqueTags := map[string]bool{}
				for _, v := range args.Tags {
					if v == "" {
						continue
					}
					uniqueTags[v] = true
					if !emoteTagRegex.MatchString(v) {
						return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SendString(fmt.Sprintf("Bad Emote Tag '%s'", v))
					}
				}

				tags = make([]string, len(uniqueTags))
				i := 0
				for k := range uniqueTags {
					tags[i] = k
					i++
				}
			}

			body := req.Body()

			// at this point we need to verify that whatever they upload is a "valid" file accepted file.
			imgType, err := containers.ToType(body)
			if err != nil {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SendString("Unknown Upload Format")
			}
			// if the file fails to be validated by the container check then its not a file type we support.
			// we then want to check some infomation about the file.
			// like the number of frames and the width and height of the images.
			frameCount := 0
			width := 0
			height := 0
			tmpPath := ""

			id := primitive.NewObjectIDFromTimestamp(time.Now())
			{
				tmp := gCtx.Config().TempFolder
				if tmp == "" {
					tmp = "tmp"
				}
				if err := os.MkdirAll(tmp, 0700); err != nil {
					logrus.WithError(err).Error("failed to create temp folder")
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}
				tmpPath = path.Join(tmp, fmt.Sprintf("%s.%s", id.Hex(), imgType))
				if err := os.WriteFile(tmpPath, body, 0600); err != nil {
					logrus.WithError(err).Error("failed to write temp file")
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}
				defer os.Remove(tmpPath)
			}

			switch imgType {
			case image.AVI, image.AVIF, image.FLV, image.MP4, image.WEBM, image.GIF, image.JPEG, image.PNG, image.TIFF:
				// use ffprobe to get the number of frames and width/height
				// ffprobe -v error -select_streams v:0 -count_frames -show_entries stream=nb_read_frames,width,height -of csv=p=0 file.ext

				output, err := exec.CommandContext(c.Context(),
					"ffprobe",
					"-v", "fatal",
					"-select_streams", "v:0",
					"-count_frames",
					"-show_entries",
					"stream=nb_read_frames,width,height",
					"-of", "csv=p=0",
					tmpPath,
				).Output()
				if err != nil {
					logrus.WithError(err).Error("failed to run ffprobe command")
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				splits := strings.Split(strings.TrimSpace(utils.B2S(output)), ",")
				if len(splits) != 3 {
					logrus.Errorf("ffprobe command returned bad results: %s", output)
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				width, err = strconv.Atoi(splits[0])
				if err != nil {
					logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				height, err = strconv.Atoi(splits[1])
				if err != nil {
					logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				frameCount, err = strconv.Atoi(splits[2])
				if err != nil {
					logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}
			case image.WEBP:
				// use a webpmux -info to get the frame count and width/height
				output, err := exec.CommandContext(c.Context(),
					"webpmux",
					"-info",
					tmpPath,
				).Output()
				if err != nil {
					logrus.WithError(err).Error("failed to run webpmux command")
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				matches := webpMuxRegex.FindAllStringSubmatch(utils.B2S(output), 1)
				if len(matches) == 0 {
					logrus.Errorf("webpmux command returned bad results: %s", output)
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				width, err = strconv.Atoi(matches[0][0])
				if err != nil {
					logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				height, err = strconv.Atoi(matches[0][1])
				if err != nil {
					logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
					return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
				}

				if matches[0][2] != "" {
					frameCount, err = strconv.Atoi(matches[0][2])
					if err != nil {
						logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
						return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
					}
				} else {
					frameCount = 1
				}
			}

			if frameCount > MAX_FRAMES {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SendString(fmt.Sprintf("Too Many Frames (got %d, but the maximum is %d)", frameCount, MAX_FRAMES))
			}

			if width > MAX_WIDTH || width <= 0 {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SendString(fmt.Sprintf("Bad Input Width (got %d, but the maximum is %d)", width, MAX_WIDTH))
			}

			if height > MAX_HEIGHT || height <= 0 {
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeBadRequest).SendString(fmt.Sprintf("Bad Input Height (got %d, but the maximum is %d", height, MAX_HEIGHT))
			}

			// Create the emote in DB
			eb := structures.NewEmoteBuilder(&structures.Emote{
				ID:         id,
				OwnerID:    actor.ID,
				Name:       name,
				Status:     structures.EmoteStatusPending,
				Tags:       tags,
				FrameCount: int32(frameCount),
				Formats:    []structures.EmoteFormat{},
			})
			if _, err = gCtx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).InsertOne(ctx, eb.Emote); err != nil {
				logrus.WithError(err).Error("mongo, failed to create pending emote in DB")
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
			}

			// at this point we are confident that the image is valid and that we can send it over to the EmoteProcessor and it will succeed.
			fileKey := fmt.Sprintf("%s.%s", id.Hex(), imgType)
			if err := gCtx.Inst().AwsS3.UploadFile(
				c.Context(),
				gCtx.Config().Aws.InternalBucket,
				fileKey,
				bytes.NewBuffer(body),
				utils.StringPointer(mime.TypeByExtension(path.Ext(tmpPath))),
				aws.AclPrivate,
				aws.DefaultCacheControl,
			); err != nil {
				logrus.WithError(err).Errorf("failed to upload image to aws")
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
			}

			providerDetails, _ := json.Marshal(job.RawProviderDetailsAws{
				Bucket: gCtx.Config().Aws.InternalBucket,
				Key:    fileKey,
			})

			consumerDetails, _ := json.Marshal(job.ResultConsumerDetailsAws{
				Bucket:    gCtx.Config().Aws.PublicBucket,
				KeyFolder: fmt.Sprintf("emote/%s", id.Hex()),
			})

			msg, _ := json.Marshal(job.Job{
				ID:                    id.Hex(),
				RawProvider:           job.AwsProvider,
				RawProviderDetails:    providerDetails,
				ResultConsumer:        job.AwsConsumer,
				ResultConsumerDetails: consumerDetails,
			})

			if err := gCtx.Inst().Rmq.Publish(gCtx.Config().Rmq.JobQueueName, "application/json", amqp.Persistent, msg); err != nil {
				logrus.WithError(err).Errorf("failed to add job to rmq")
				return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeInternalServerError).SendString("Internal Server Error")
			}

			// validate this data
			j, _ := json.Marshal(map[string]string{"id": eb.Emote.ID.Hex()})
			return helpers.HttpResponse(c).SetStatus(helpers.HttpStatusCodeCreated).Send(j)
		},
	)
}

type CreateEmoteData struct {
	Name  string               `json:"name"`
	Tags  [MAX_TAGS]string     `json:"tags"`
	Flags structures.EmoteFlag `json:"flags"`
}
