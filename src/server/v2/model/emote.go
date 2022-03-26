package model

import (
	"fmt"

	v2structures "github.com/SevenTV/Common/structures/v2"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
	"github.com/SevenTV/REST/src/global"
)

type Emote struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Owner            *User       `json:"owner"`
	Visibility       int32       `json:"visibility"`
	VisibilitySimple []string    `json:"visibility_simple"`
	Mime             string      `json:"mime"`
	Status           int8        `json:"status"`
	Tags             []string    `json:"tags"`
	Width            []int32     `json:"width"`
	Height           []int32     `json:"height"`
	URLs             [][2]string `json:"urls"`
}

func NewEmote(ctx global.Context, s *structures.Emote) *Emote {
	version, _ := s.GetVersion(s.ID)
	width := make([]int32, 4)
	height := make([]int32, 4)
	urls := make([][2]string, 4)
	if version != nil {
		for _, format := range version.Formats {
			if format.Name != structures.EmoteFormatNameWEBP {
				continue
			}
			pos := 0
			for _, f := range format.Files {
				if version.FrameCount > 1 && !f.Animated || pos > 4 {
					continue
				}

				width[pos] = f.Width
				height[pos] = f.Height
				urls[pos] = [2]string{
					fmt.Sprintf("%dx", pos+1),
					fmt.Sprintf("//%s/emote/%s/%s", ctx.Config().CdnURL, version.ID.Hex(), f.Name),
				}
				pos++
			}
		}
	}

	vis := 0
	if !version.State.Listed {
		vis |= int(v2structures.EmoteVisibilityUnlisted)
	}
	if utils.BitField.HasBits(int64(s.Flags), int64(structures.EmoteFlagsZeroWidth)) {
		vis |= int(v2structures.EmoteVisibilityZeroWidth)
	}
	if utils.BitField.HasBits(int64(s.Flags), int64(structures.EmoteFlagsPrivate)) {
		vis |= int(v2structures.EmoteVisibilityPrivate)
	}

	owner := structures.DeletedUser
	if s.Owner != nil {
		owner = s.Owner
	}

	return &Emote{
		ID:               s.ID.Hex(),
		Name:             s.Name,
		Owner:            NewUser(owner),
		Visibility:       int32(vis),
		VisibilitySimple: []string{},
		Mime:             "",
		Status:           0,
		Tags:             []string{},
		Width:            width,
		Height:           height,
		URLs:             [][2]string{},
	}
}
