package emotes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SevenTV/Common/errors"
	"github.com/SevenTV/Common/mongo"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/aws"
	"github.com/SevenTV/REST/src/global"
	"github.com/SevenTV/REST/src/server/rest"
	"github.com/SevenTV/REST/src/server/v3/middleware"
	"github.com/seventv/EmoteProcessor/src/containers"
	"github.com/seventv/EmoteProcessor/src/image"
	"github.com/seventv/EmoteProcessor/src/job"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type create struct {
	Ctx global.Context
}

func newCreate(gCtx global.Context) rest.Route {
	return &create{gCtx}
}

func (r *create) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "",
		Method: rest.POST,
		Middleware: []rest.Middleware{
			middleware.Auth(r.Ctx),
		},
	}
}

func (r *create) Handler(ctx *rest.Ctx) rest.APIError {
	ctx.SetContentType("application/json")

	// Check RMQ status
	if r.Ctx.Inst().Rmq == nil {
		return errors.ErrMissingInternalDependency().SetDetail("Emote Processing Service Unavailable")
	}

	// Get actor
	actor, ok := ctx.GetActor()
	if !ok {
		return errors.ErrUnauthorized()
	}

	if !actor.HasPermission(structures.RolePermissionCreateEmote) {
		return errors.ErrInsufficientPrivilege()
	}

	req := ctx.Request
	var (
		name  string
		tags  []string
		flags structures.EmoteFlag
	)

	// these validations are all "free" as in we can do them before we download the file they try to upload.
	args := &createData{}
	if err := json.Unmarshal(req.Header.Peek("X-Emote-Data"), args); err != nil {
		return errors.ErrInvalidRequest().SetDetail(err.Error())
	}

	// Validate: Name
	{
		if !emoteNameRegex.MatchString(args.Name) {
			return errors.ErrInvalidRequest().SetDetail("Bad Emote Name")
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
				return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Bad Emote Tag '%s'", v))
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
	if false {
		return ctx.JSON(200, map[string]string{"very": "nice"})
	}

	// at this point we need to verify that whatever they upload is a "valid" file accepted file.
	imgType, err := containers.ToType(body)
	if err != nil {
		return errors.ErrInvalidRequest().SetDetail("Unknown Upload Format")
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
		tmp := r.Ctx.Config().TempFolder
		if tmp == "" {
			tmp = "tmp"
		}
		if err := os.MkdirAll(tmp, 0700); err != nil {
			logrus.WithError(err).Error("failed to create temp folder")
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}
		tmpPath = path.Join(tmp, fmt.Sprintf("%s.%s", id.Hex(), imgType))
		if err := os.WriteFile(tmpPath, body, 0600); err != nil {
			logrus.WithError(err).Error("failed to write temp file")
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}
		defer os.Remove(tmpPath)
	}

	switch imgType {
	case image.AVI, image.AVIF, image.FLV, image.MP4, image.WEBM, image.GIF, image.JPEG, image.PNG, image.TIFF:
		// use ffprobe to get the number of frames and width/height
		// ffprobe -v error -select_streams v:0 -count_frames -show_entries stream=nb_read_frames,width,height -of csv=p=0 file.ext

		output, err := exec.CommandContext(ctx,
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
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		splits := strings.Split(strings.TrimSpace(utils.B2S(output)), ",")
		if len(splits) != 3 {
			logrus.Errorf("ffprobe command returned bad results: %s", output)
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		width, err = strconv.Atoi(splits[0])
		if err != nil {
			logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		height, err = strconv.Atoi(splits[1])
		if err != nil {
			logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		frameCount, err = strconv.Atoi(splits[2])
		if err != nil {
			logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}
	case image.WEBP:
		// use a webpmux -info to get the frame count and width/height
		output, err := exec.CommandContext(ctx,
			"webpmux",
			"-info",
			tmpPath,
		).Output()
		if err != nil {
			logrus.WithError(err).Error("failed to run webpmux command")
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		matches := webpMuxRegex.FindAllStringSubmatch(utils.B2S(output), 1)
		if len(matches) == 0 {
			logrus.Errorf("webpmux command returned bad results: %s", output)
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		width, err = strconv.Atoi(matches[0][1])
		if err != nil {
			logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		height, err = strconv.Atoi(matches[0][2])
		if err != nil {
			logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
			return errors.ErrInternalServerError().SetDetail("Internal Server Error")
		}

		if matches[0][3] != "" {
			frameCount, err = strconv.Atoi(matches[0][3])
			if err != nil {
				logrus.WithError(err).Errorf("ffprobe command returned bad results: %s", output)
				return errors.ErrInternalServerError().SetDetail("Internal Server Error")
			}
		} else {
			frameCount = 1
		}
	}

	if frameCount > MAX_FRAMES {
		return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Too Many Frames (got %d, but the maximum is %d)", frameCount, MAX_FRAMES))
	}

	if width > MAX_WIDTH || width <= 0 {
		return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Bad Input Width (got %d, but the maximum is %d)", width, MAX_WIDTH))
	}

	if height > MAX_HEIGHT || height <= 0 {
		return errors.ErrInvalidRequest().SetDetail(fmt.Sprintf("Bad Input Height (got %d, but the maximum is %d", height, MAX_HEIGHT))
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
		Flags:      flags,
	})
	if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameEmotes).InsertOne(ctx, eb.Emote); err != nil {
		logrus.WithError(err).Error("mongo, failed to create pending emote in DB")
		return errors.ErrInternalServerError().SetDetail("Internal Server Error")
	}

	// at this point we are confident that the image is valid and that we can send it over to the EmoteProcessor and it will succeed.
	fileKey := fmt.Sprintf("%s.%s", id.Hex(), imgType)
	internalFilekey := fmt.Sprintf("internal/emote/%s", fileKey)
	if err := r.Ctx.Inst().AwsS3.UploadFile(
		ctx,
		r.Ctx.Config().Aws.Bucket,
		internalFilekey,
		bytes.NewBuffer(body),
		utils.StringPointer(mime.TypeByExtension(path.Ext(tmpPath))),
		aws.AclPrivate,
		aws.DefaultCacheControl,
	); err != nil {
		logrus.WithError(err).Errorf("failed to upload image to aws")
		return errors.ErrInternalServerError().SetDetail("Internal Server Error")
	}

	providerDetails, _ := json.Marshal(job.RawProviderDetailsAws{
		Bucket: r.Ctx.Config().Aws.Bucket,
		Key:    internalFilekey,
	})

	consumerDetails, _ := json.Marshal(job.ResultConsumerDetailsAws{
		Bucket:    r.Ctx.Config().Aws.Bucket,
		KeyFolder: fmt.Sprintf("emote/%s", id.Hex()),
	})

	msg, _ := json.Marshal(&job.Job{
		ID:                    id.Hex(),
		RawProvider:           job.AwsProvider,
		RawProviderDetails:    providerDetails,
		ResultConsumer:        job.AwsConsumer,
		ResultConsumerDetails: consumerDetails,
	})

	if err := r.Ctx.Inst().Rmq.Publish(r.Ctx.Config().Rmq.JobQueueName, "application/json", amqp.Persistent, msg); err != nil {
		logrus.WithError(err).Errorf("failed to add job to rmq")
		return errors.ErrInternalServerError().SetDetail("Internal Server Error")
	}

	return ctx.JSON(rest.Created, map[string]string{"id": eb.Emote.ID.Hex()})
}

type createData struct {
	Name  string               `json:"name"`
	Tags  [MAX_TAGS]string     `json:"tags"`
	Flags structures.EmoteFlag `json:"flags"`
}

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
	webpMuxRegex   = regexp.MustCompile(`Canvas size: (\d+) x (\d+)(?:\n?.*\n){0,2}(?:Number of frames: (\d+))?`) // capture group 1: width, 2: height, 3: frame count or empty which means 1
)
